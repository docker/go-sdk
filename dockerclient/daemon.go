package dockerclient

import (
	"context"
	"errors"
	"net/url"
	"os"

	"github.com/docker/docker/api/types/network"
)

// dockerEnvFile is the file that is created when running inside a container.
// It's a variable to allow testing.
var dockerEnvFile = "/.dockerenv"

func (c *Client) DaemonHost(ctx context.Context) (string, error) {
	// infer from Docker host
	daemonURL, err := url.Parse(c.client.DaemonHost())
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
				ip = "localhost"
			}
			host = ip
		} else {
			host = "localhost"
		}
	default:
		return "", errors.New("could not determine host through env or docker host")
	}

	return host, nil
}

func (c *Client) getGatewayIP(ctx context.Context, defaultNetwork string) (string, error) {
	nw, err := c.client.NetworkInspect(ctx, defaultNetwork, network.InspectOptions{})
	if err != nil {
		return "", err
	}

	var ip string
	for _, cfg := range nw.IPAM.Config {
		if cfg.Gateway != "" {
			ip = cfg.Gateway
			break
		}
	}
	if ip == "" {
		return "", errors.New("failed to get gateway IP from network settings")
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
