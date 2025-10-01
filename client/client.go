package client

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"path/filepath"
	"time"

	"github.com/docker/docker/client"
	dockercontext "github.com/docker/go-sdk/context"
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

	defaultHealthCheck = func(ctx context.Context) func(c SDKClient) error {
		return func(c SDKClient) error {
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
//	cli, err := client.New(context.Background(), client.FromDockerOpt(client.WithHost("tcp://foobar:2375")))
//
// The client uses a logger that is initialized to [io.Discard]; you can change it by
// providing the [WithLogger] option.
// E.g.
//
//	cli, err := client.New(context.Background(), client.WithLogger(slog.Default()))
//
// The client is safe for concurrent use by multiple goroutines.
func New(ctx context.Context, options ...ClientOption) (SDKClient, error) {
	c := &sdkClient{
		log:         defaultLogger,
		healthCheck: defaultHealthCheck,
	}
	for _, opt := range options {
		if err := opt.Apply(c); err != nil {
			return nil, fmt.Errorf("apply option: %w", err)
		}
	}

	if err := c.init(); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	if err := c.healthCheck(ctx)(c); err != nil {
		return nil, fmt.Errorf("health check: %w", err)
	}

	return c, nil
}

// init initializes the client.
// This method is safe for concurrent use by multiple goroutines.
func (c *sdkClient) init() error {
	if c.APIClient != nil || c.err != nil {
		return c.err
	}

	// Set the default values for the client:
	// - log
	// - dockerHost
	// - currentContext
	if c.err = c.defaultValues(); c.err != nil {
		return fmt.Errorf("default values: %w", c.err)
	}

	if c.cfg, c.err = newConfig(c.dockerHost); c.err != nil {
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
	maps.Copy(httpHeaders, c.extraHeaders)

	// Append the SDK headers last.
	httpHeaders[headerUserAgent] = defaultUserAgent

	opts = append(opts, client.WithHTTPHeaders(httpHeaders))

	api, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return fmt.Errorf("new client: %w", err)
	}
	c.APIClient = api
	return nil
}

// defaultValues sets the default values for the client.
// If no logger is provided, the default one is used.
// If no docker host is provided and no docker context is provided, the current docker host and context are used.
// If no docker host is provided but a docker context is provided, the docker host from the context is used.
// If a docker host is provided, it is used as is.
func (c *sdkClient) defaultValues() error {
	if c.log == nil {
		c.log = defaultLogger
	}

	if c.dockerHost == "" && c.dockerContext == "" {
		currentDockerHost, err := dockercontext.CurrentDockerHost()
		if err != nil {
			return fmt.Errorf("current docker host: %w", err)
		}
		currentContext, err := dockercontext.Current()
		if err != nil {
			return fmt.Errorf("current context: %w", err)
		}

		c.dockerHost = currentDockerHost
		c.dockerContext = currentContext

		return nil
	}

	if c.dockerContext != "" {
		dockerHost, err := dockercontext.DockerHostFromContext(c.dockerContext)
		if err != nil {
			return fmt.Errorf("docker host from context: %w", err)
		}

		c.dockerHost = dockerHost
	}

	return nil
}
