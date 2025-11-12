package container

import (
	"context"
	"net/netip"
)

// ContainerIP gets the IP address of the primary network within the container.
// If there are multiple networks, it returns an empty string.
func (c *Container) ContainerIP(ctx context.Context) (netip.Addr, error) {
	inspect, err := c.Inspect(ctx)
	if err != nil {
		return netip.Addr{}, err
	}

	// use IP from "Networks" if only single network defined
	var ip netip.Addr
	networks := inspect.Container.NetworkSettings.Networks
	if len(networks) == 1 {
		for _, v := range networks {
			ip = v.IPAddress
		}
	}
	return ip, nil
}

// ContainerIPs gets the IP addresses of all the networks within the container.
func (c *Container) ContainerIPs(ctx context.Context) ([]netip.Addr, error) {
	inspect, err := c.Inspect(ctx)
	if err != nil {
		return nil, err
	}

	ips := make([]netip.Addr, 0, len(inspect.Container.NetworkSettings.Networks))
	for _, nw := range inspect.Container.NetworkSettings.Networks {
		ips = append(ips, nw.IPAddress)
	}

	return ips, nil
}

// NetworkAliases gets the aliases of the container for the networks it is attached to.
func (c *Container) NetworkAliases(ctx context.Context) (map[string][]string, error) {
	inspect, err := c.Inspect(ctx)
	if err != nil {
		return map[string][]string{}, err
	}

	networks := inspect.Container.NetworkSettings.Networks

	a := map[string][]string{}

	for k := range networks {
		a[k] = networks[k].Aliases
	}

	return a, nil
}

// Networks gets the names of the networks the container is attached to.
func (c *Container) Networks(ctx context.Context) ([]string, error) {
	inspect, err := c.Inspect(ctx)
	if err != nil {
		return []string{}, err
	}

	networks := inspect.Container.NetworkSettings.Networks

	var n []string

	for k := range networks {
		n = append(n, k)
	}

	return n, nil
}
