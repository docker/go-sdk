package dockerclient

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/client"
	"github.com/docker/go-sdk/dockercontext"
)

// packagePath is the package path for the docker-go-sdk package.
const packagePath = "github.com/docker/go-sdk"

// DefaultClient is the default client for interacting with containers.
var DefaultClient = &Client{}

// Client is a type that represents a client for interacting with containers.
type Client struct {
	// log is the logger for the client.
	log *slog.Logger

	// mtx is a mutex for synchronizing access to the fields below.
	mtx sync.RWMutex

	// client is the underlying docker client.
	client *client.Client

	// cfg is the configuration for the client, obtained from the environment variables.
	cfg *config

	// err is used to store errors that occur during the client's initialization.
	err error

	// dockerOpts are options to be passed to the docker client.
	dockerOpts []client.Opt

	// extraHeaders are additional headers to be sent to the docker client.
	extraHeaders map[string]string

	// cached docker info
	dockerInfo    system.Info
	dockerInfoSet bool

	// healthCheck is a function that returns the health of the docker daemon.
	// If not set, the default health check will be used.
	healthCheck func(ctx context.Context) func(c *Client) error
}

// implements SystemAPIClient interface
var _ client.SystemAPIClient = &Client{}

// Events returns a channel to listen to events that happen to the docker daemon.
func (c *Client) Events(ctx context.Context, options events.ListOptions) (<-chan events.Message, <-chan error) {
	return c.client.Events(ctx, options)
}

// Info returns information about the docker server. The result of Info is cached
// and reused every time Info is called.
// It will also print out the docker server info, and the resolved Docker paths, to the default logger.
func (c *Client) Info(ctx context.Context) (system.Info, error) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	if c.dockerInfoSet {
		return c.dockerInfo, nil
	}

	info, err := c.client.Info(ctx)
	if err != nil {
		return info, fmt.Errorf("docker info: %w", err)
	}
	c.dockerInfo = info
	c.dockerInfoSet = true

	infoLabels := ""
	if len(c.dockerInfo.Labels) > 0 {
		infoLabels = `
  Labels:`
		for _, lb := range c.dockerInfo.Labels {
			infoLabels += "\n    " + lb
		}
	}

	currentContext, err := dockercontext.Current()
	if err != nil {
		return c.dockerInfo, fmt.Errorf("current context: %w", err)
	}

	dockerHost, err := dockercontext.CurrentDockerHost()
	if err != nil {
		return c.dockerInfo, fmt.Errorf("current docker host: %w", err)
	}

	c.log.Info("Connected to docker",
		"package", packagePath,
		"server_version", c.dockerInfo.ServerVersion,
		"client_version", c.client.ClientVersion(),
		"operating_system", c.dockerInfo.OperatingSystem,
		"mem_total", c.dockerInfo.MemTotal/1024/1024,
		"labels", infoLabels,
		"current_context", currentContext,
		"docker_host", dockerHost,
	)

	return c.dockerInfo, nil
}

// RegistryLogin logs into a Docker registry.
func (c *Client) RegistryLogin(ctx context.Context, auth registry.AuthConfig) (registry.AuthenticateOKBody, error) {
	return c.client.RegistryLogin(ctx, auth)
}

// DiskUsage returns the disk usage of all images.
func (c *Client) DiskUsage(ctx context.Context, options types.DiskUsageOptions) (types.DiskUsage, error) {
	return c.client.DiskUsage(ctx, options)
}

// Ping pings the docker server.
func (c *Client) Ping(ctx context.Context) (types.Ping, error) {
	return c.client.Ping(ctx)
}
