package container

import "context"

// ContainerIP gets the IP address of the primary network within the container.
// If there are multiple networks, it returns an empty string.
func (c *Container) ContainerIP(ctx context.Context) (string, error) {
	inspect, err := c.Inspect(ctx)
	if err != nil {
		return "", err
	}

	ip := inspect.NetworkSettings.IPAddress
	if ip == "" {
		// use IP from "Networks" if only single network defined
		networks := inspect.NetworkSettings.Networks
		if len(networks) == 1 {
			for _, v := range networks {
				ip = v.IPAddress
			}
		}
	}

	return ip, nil
}

// ContainerIPs gets the IP addresses of all the networks within the container.
func (c *Container) ContainerIPs(ctx context.Context) ([]string, error) {
	ips := make([]string, 0)

	inspect, err := c.Inspect(ctx)
	if err != nil {
		return nil, err
	}

	networks := inspect.NetworkSettings.Networks
	for _, nw := range networks {
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

	networks := inspect.NetworkSettings.Networks

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

	networks := inspect.NetworkSettings.Networks

	n := []string{}

	for k := range networks {
		n = append(n, k)
	}

	return n, nil
}
