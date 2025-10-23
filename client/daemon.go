package client

import (
	"context"
	"errors"
	"net/netip"
	"net/url"
	"os"

	"github.com/moby/moby/client"
)

// dockerEnvFile is the file that is created when running inside a container.
// It's a variable to allow testing.
var dockerEnvFile = "/.dockerenv"

// DaemonHostWithContext gets the host or ip of the Docker daemon where ports are exposed on
// Warning: this is based on your Docker host setting. Will fail if using an SSH tunnel
func (c *sdkClient) DaemonHostWithContext(ctx context.Context) (string, error) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	return c.daemonHostLocked(ctx)
}

func (c *sdkClient) daemonHostLocked(ctx context.Context) (string, error) {
	// infer from Docker host
	daemonURL, err := url.Parse(c.DaemonHost())
	if err != nil {
		return "", err
	}

	var host string

	switch daemonURL.Scheme {
	case "http", "https", "tcp":
		host = daemonURL.Hostname()
	case "unix", "npipe":
		if inAContainer(dockerEnvFile) {
			ip, err := c.getGatewayIP(ctx, "bridge")
			if err != nil {
				host = "localhost"
			} else {
				host = ip.String()
			}
		} else {
			host = "localhost"
		}
	default:
		return "", errors.New("could not determine host through env or docker host")
	}

	return host, nil
}

func (c *sdkClient) getGatewayIP(ctx context.Context, defaultNetwork string) (netip.Addr, error) {
	nw, err := c.NetworkInspect(ctx, defaultNetwork, client.NetworkInspectOptions{})
	if err != nil {
		return netip.Addr{}, err
	}

	var ip netip.Addr
	for _, cfg := range nw.Network.IPAM.Config {
		if cfg.Gateway.IsValid() {
			ip = cfg.Gateway
			break
		}
	}
	if !ip.IsValid() {
		return netip.Addr{}, errors.New("failed to get gateway IP from network settings")
	}

	return ip, nil
}

// InAContainer returns true if the code is running inside a container
// See https://github.com/docker/docker/blob/a9fa38b1edf30b23cae3eade0be48b3d4b1de14b/daemon/initlayer/setup_unix.go#L25
func inAContainer(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}
