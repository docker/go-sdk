package container

import (
	"context"
	"fmt"
	"net"

	"github.com/containerd/errdefs"

	"github.com/docker/go-connections/nat"
)

// Endpoint gets proto://host:port string for the lowest numbered exposed port
// Will return just host:port if proto is empty
func (c *Container) Endpoint(ctx context.Context, proto string) (string, error) {
	inspect, err := c.Inspect(ctx)
	if err != nil {
		return "", err
	}

	if len(inspect.NetworkSettings.Ports) == 0 {
		return "", errdefs.ErrNotFound.WithMessage("no ports exposed")
	}

	// Get lowest numbered bound port.
	var lowestPort nat.Port
	for port := range inspect.NetworkSettings.Ports {
		if lowestPort == "" || port.Int() < lowestPort.Int() {
			lowestPort = port
		}
	}

	return c.PortEndpoint(ctx, lowestPort, proto)
}

// PortEndpoint gets proto://host:port string for the given exposed port
// It returns proto://host:port or proto://[IPv6host]:port string for the given exposed port.
// It returns just host:port or [IPv6host]:port if proto is blank.
func (c *Container) PortEndpoint(ctx context.Context, port nat.Port, proto string) (string, error) {
	host, err := c.Host(ctx)
	if err != nil {
		return "", err
	}

	outerPort, err := c.MappedPort(ctx, port)
	if err != nil {
		return "", err
	}

	hostPort := net.JoinHostPort(host, outerPort.Port())
	if proto == "" {
		return hostPort, nil
	}

	return proto + "://" + hostPort, nil
}

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
