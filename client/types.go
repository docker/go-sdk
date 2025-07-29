package client

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/client"
)

// packagePath is the package path for the docker-go-sdk package.
const packagePath = "github.com/docker/go-sdk"

// SDKClient extends SDKClient with higher-level functions
type SDKClient interface {
	client.APIClient

	// Logger returns the logger for the client.
	Logger() *slog.Logger

	// DaemonHost gets the host or ip of the Docker daemon where ports are exposed on
	DaemonHostWithContext(ctx context.Context) (string, error)

	// FindContainerByName finds a container by name.
	FindContainerByName(ctx context.Context, name string) (*container.Summary, error)
}

var _ client.APIClient = &sdkClient{}

// sdkClient is a type that represents a client for interacting with containers.
type sdkClient struct {
	client.APIClient

	// log is the logger for the client.
	log *slog.Logger

	// mtx is a mutex for synchronizing access to the fields below.
	mtx sync.RWMutex

	// cfg is the configuration for the client, obtained from the environment variables.
	cfg *config

	// err is used to store errors that occur during the client's initialization.
	err error

	// dockerOpts are options to be passed to the docker client.
	dockerOpts []client.Opt

	// dockerContext is the current context of the docker daemon.
	dockerContext string

	// dockerHost is the host of the docker daemon.
	dockerHost string

	// extraHeaders are additional headers to be sent to the docker client.
	extraHeaders map[string]string

	// cached docker info
	dockerInfo    system.Info
	dockerInfoSet bool

	// healthCheck is a function that returns the health of the docker daemon.
	// If not set, the default health check will be used.
	healthCheck func(ctx context.Context) func(c SDKClient) error
}

// Logger returns the logger for the client.
func (c *sdkClient) Logger() *slog.Logger {
	return c.log
}

// Info returns information about the docker server. The result of Info is cached
// and reused every time Info is called.
// It will also print out the docker server info, and the resolved Docker paths, to the default logger.
func (c *sdkClient) Info(ctx context.Context) (system.Info, error) {
	c.mtx.Lock()
	if c.dockerInfoSet {
		defer c.mtx.Unlock()
		return c.dockerInfo, nil
	}
	c.mtx.Unlock()

	var info system.Info

	info, err := c.APIClient.Info(ctx)
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

	c.log.Info("Connected to docker",
		"package", packagePath,
		"server_version", c.dockerInfo.ServerVersion,
		"client_version", c.ClientVersion(),
		"operating_system", c.dockerInfo.OperatingSystem,
		"mem_total", c.dockerInfo.MemTotal/1024/1024,
		"labels", infoLabels,
		"docker_context", c.dockerContext,
		"docker_host", c.dockerHost,
	)

	return c.dockerInfo, nil
}
