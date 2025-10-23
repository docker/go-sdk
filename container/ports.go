package container

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/network"
)

// Endpoint gets proto://host:port string for the lowest numbered exposed port
// Will return just host:port if proto is empty
func (c *Container) Endpoint(ctx context.Context, proto string) (string, error) {
	inspect, err := c.Inspect(ctx)
	if err != nil {
		return "", err
	}

	if len(inspect.Container.NetworkSettings.Ports) == 0 {
		return "", errdefs.ErrNotFound.WithMessage("no ports exposed")
	}

	// Get lowest numbered bound port.
	var lowestPort network.Port
	for port := range inspect.Container.NetworkSettings.Ports {
		if lowestPort.IsZero() || port.Num() < lowestPort.Num() {
			lowestPort = port
		}
	}

	return c.PortEndpoint(ctx, lowestPort, proto)
}

// PortEndpoint gets proto://host:port string for the given exposed port
// It returns proto://host:port or proto://[IPv6host]:port string for the given exposed port.
// It returns just host:port or [IPv6host]:port if proto is blank.
//
// TODO(robmry) - remove proto and use port.Proto()
func (c *Container) PortEndpoint(ctx context.Context, port network.Port, proto string) (string, error) {
	host, err := c.Host(ctx)
	if err != nil {
		return "", err
	}

	outerPort, err := c.MappedPort(ctx, port)
	if err != nil {
		return "", err
	}

	hostPort := net.JoinHostPort(host, strconv.Itoa(int(outerPort.Num())))
	if proto == "" {
		return hostPort, nil
	}

	return proto + "://" + hostPort, nil
}

// MappedPort gets externally mapped port for a container port
func (c *Container) MappedPort(ctx context.Context, port network.Port) (network.Port, error) {
	inspect, err := c.Inspect(ctx)
	if err != nil {
		return network.Port{}, fmt.Errorf("inspect: %w", err)
	}
	if inspect.Container.HostConfig.NetworkMode == "host" {
		return port, nil
	}

	ports := inspect.Container.NetworkSettings.Ports

	for k, p := range ports {
		if k != port {
			continue
		}
		if port.Proto() != "" && k.Proto() != port.Proto() {
			continue
		}
		if len(p) == 0 {
			continue
		}
		return network.ParsePort(p[0].HostPort + "/" + string(k.Proto()))
	}

	return network.Port{}, errdefs.ErrNotFound.WithMessage(fmt.Sprintf("port %q not found", port))
}
