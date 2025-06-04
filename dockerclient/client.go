package dockerclient

import (
	"context"
	"fmt"
	"path/filepath"

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

	// packagePath is the package path for the docker-go-sdk package.
	packagePath = "github.com/docker/go-sdk"
)

// New returns a new client for interacting with containers.
func New(ctx context.Context, options ...ClientOption) (*Client, error) {
	client := &Client{}
	for _, opt := range options {
		if err := opt.Apply(client); err != nil {
			return nil, err
		}
	}

	if err := client.initOnce(ctx); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	return client, nil
}

// initOnce initializes the client once.
// This method is safe for concurrent use by multiple goroutines.
func (c *Client) initOnce(_ context.Context) error {
	c.mtx.RLock()
	if c.client != nil || c.err != nil {
		err := c.err
		c.mtx.RUnlock()
		return err
	}
	c.mtx.RUnlock()

	c.mtx.Lock()
	defer c.mtx.Unlock()

	if c.cfg, c.err = newConfig(); c.err != nil {
		return c.err
	}

	opts := []client.Opt{client.FromEnv, client.WithAPIVersionNegotiation()}

	dockerHost, err := dockercontext.CurrentDockerHost()
	if err != nil {
		return fmt.Errorf("current docker host: %w", err)
	}

	// Always add the resolved docker host to the client options,
	// as it cannot be empty.
	opts = append(opts, client.WithHost(dockerHost))

	if c.cfg.TLSVerify {
		// For further information see:
		// https://docs.docker.com/engine/security/protect-access/#use-tls-https-to-protect-the-docker-daemon-socket
		opts = append(opts, client.WithTLSClientConfig(
			filepath.Join(c.cfg.CertPath, tlsCACertFile),
			filepath.Join(c.cfg.CertPath, tlsCertFile),
			filepath.Join(c.cfg.CertPath, tlsKeyFile),
		))
	}

	httpHeaders := make(map[string]string)
	for k, v := range c.extraHeaders {
		httpHeaders[k] = v
	}

	// Append the SDK headers last.
	httpHeaders[headerUserAgent] = "docker-go-sdk/" + Version()

	opts = append(opts, client.WithHTTPHeaders(httpHeaders))

	if c.client, c.err = client.NewClientWithOpts(opts...); c.err != nil {
		c.err = fmt.Errorf("new client: %w", c.err)
		return c.err
	}

	return nil
}

// Close closes the client.
// This method is safe for concurrent use by multiple goroutines.
func (c *Client) Close() error {
	// Change from RLock to Lock since we're performing a write operation
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if c.client == nil {
		return nil
	}

	// Store the error before clearing the client
	err := c.client.Close()

	// Clear the client after closing to prevent use-after-close issues
	c.client = nil
	c.dockerInfo = system.Info{}
	c.dockerInfoSet = false

	return err
}

// Client returns the underlying docker client.
// This method is safe for concurrent use by multiple goroutines.
func (c *Client) Client() *client.Client {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	return c.client
}
