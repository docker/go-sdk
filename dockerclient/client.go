package dockerclient

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/client"
	"github.com/docker/go-sdk/dockercontext"
)

const (
	// Headers used for docker client requests.
	headerUserAgent = "User-Agent"

	// TLS certificate files.
	tlsCACertFile = "ca.pem"
	tlsCertFile   = "cert.pem"
	tlsKeyFile    = "key.pem"
)

var (
	defaultLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

	defaultUserAgent = "docker-go-sdk/" + Version()

	defaultOpts = []client.Opt{client.FromEnv, client.WithAPIVersionNegotiation()}

	defaultHealthCheck = func(ctx context.Context) func(c *Client) error {
		return func(c *Client) error {
			var pingErr error
			for i := range 3 {
				if _, pingErr = c.Ping(ctx); pingErr == nil {
					return nil
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(time.Millisecond * time.Duration(i+1) * 100):
				}
			}
			return fmt.Errorf("docker daemon not ready: %w", pingErr)
		}
	}
)

// New returns a new client for interacting with containers.
// The client is configured using the provided options, that must be compatible with
// docker's [client.Opt] type.
//
// The Docker host is automatically resolved reading it from the current docker context;
// in case you need to pass [client.Opt] options that override the docker host, you can
// do so by providing the [FromDockerOpt] options adapter.
// E.g.
//
//	cli, err := dockerclient.New(context.Background(), dockerclient.FromDockerOpt(client.WithHost("tcp://foobar:2375")))
//
// The client uses a logger that is initialized to [io.Discard]; you can change it by
// providing the [WithLogger] option.
// E.g.
//
//	cli, err := dockerclient.New(context.Background(), dockerclient.WithLogger(slog.Default()))
//
// The client is safe for concurrent use by multiple goroutines.
func New(ctx context.Context, options ...ClientOption) (*Client, error) {
	client := &Client{
		healthCheck: defaultHealthCheck,
	}
	for _, opt := range options {
		if err := opt.Apply(client); err != nil {
			return nil, fmt.Errorf("apply option: %w", err)
		}
	}

	if err := client.initOnce(ctx); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	if err := client.healthCheck(ctx)(client); err != nil {
		return nil, fmt.Errorf("health check: %w", err)
	}

	return client, nil
}

// initOnce initializes the client once.
// This method is safe for concurrent use by multiple goroutines.
func (c *Client) initOnce(_ context.Context) error {
	c.mtx.RLock()
	if c.Client != nil || c.err != nil {
		err := c.err
		c.mtx.RUnlock()
		return err
	}
	c.mtx.RUnlock()

	c.mtx.Lock()
	defer c.mtx.Unlock()

	if c.log == nil {
		c.log = defaultLogger
	}

	dockerHost, err := dockercontext.CurrentDockerHost()
	if err != nil {
		return fmt.Errorf("current docker host: %w", err)
	}

	if c.cfg, c.err = newConfig(dockerHost); c.err != nil {
		return c.err
	}

	opts := make([]client.Opt, len(defaultOpts), len(defaultOpts)+len(c.dockerOpts))
	copy(opts, defaultOpts)

	// Add all collected Docker options
	opts = append(opts, c.dockerOpts...)

	if c.cfg.TLSVerify {
		// For further information see:
		// https://docs.docker.com/engine/security/protect-access/#use-tls-https-to-protect-the-docker-daemon-socket
		opts = append(opts, client.WithTLSClientConfig(
			filepath.Join(c.cfg.CertPath, tlsCACertFile),
			filepath.Join(c.cfg.CertPath, tlsCertFile),
			filepath.Join(c.cfg.CertPath, tlsKeyFile),
		))
	}
	if c.cfg.Host != "" {
		// apply the host from the config if it is set
		opts = append(opts, client.WithHost(c.cfg.Host))
	}

	httpHeaders := make(map[string]string)
	for k, v := range c.extraHeaders {
		httpHeaders[k] = v
	}

	// Append the SDK headers last.
	httpHeaders[headerUserAgent] = defaultUserAgent

	opts = append(opts, client.WithHTTPHeaders(httpHeaders))

	if c.Client, c.err = client.NewClientWithOpts(opts...); c.err != nil {
		c.err = fmt.Errorf("new client: %w", c.err)
		return c.err
	}

	return nil
}

// Close closes the client.
// This method is safe for concurrent use by multiple goroutines.
func (c *Client) Close() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if c.Client == nil {
		return nil
	}

	// Store the error before clearing the client
	err := c.Client.Close()

	// Clear the client after closing to prevent use-after-close issues
	c.dockerInfo = system.Info{}
	c.dockerInfoSet = false

	return err
}

// Logger returns the logger for the client.
// This method is safe for concurrent use by multiple goroutines.
func (c *Client) Logger() *slog.Logger {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.log
}
