package dockercontainer

import (
	"context"
	"fmt"

	"github.com/containerd/errdefs"

	"github.com/docker/go-connections/nat"
)

// MappedPort gets externally mapped port for a container port
func (c *Container) MappedPort(ctx context.Context, port nat.Port) (nat.Port, error) {
	inspect, err := c.Inspect(ctx)
	if err != nil {
		return "", fmt.Errorf("inspect: %w", err)
	}
	if inspect.HostConfig.NetworkMode == "host" {
		return port, nil
	}

	ports := inspect.NetworkSettings.Ports

	for k, p := range ports {
		if k.Port() != port.Port() {
			continue
		}
		if port.Proto() != "" && k.Proto() != port.Proto() {
			continue
		}
		if len(p) == 0 {
			continue
		}
		return nat.NewPort(k.Proto(), p[0].HostPort)
	}

	return "", errdefs.ErrNotFound.WithMessage(fmt.Sprintf("port %q not found", port))
}
