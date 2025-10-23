package network

import (
	dockerclient "github.com/moby/moby/client"

	"github.com/docker/go-sdk/client"
)

// Network represents a Docker network.
type Network struct {
	response     dockerclient.NetworkCreateResult
	inspect      dockerclient.NetworkInspectResult
	dockerClient client.SDKClient
	opts         *options
	name         string
}

// ID returns the ID of the network.
func (n *Network) ID() string {
	return n.response.ID
}

// Driver returns the driver of the network.
func (n *Network) Driver() string {
	return n.opts.driver
}

// Name returns the name of the network.
func (n *Network) Name() string {
	return n.name
}

// Client returns the client used to create the network.
func (n *Network) Client() client.SDKClient {
	return n.dockerClient
}
