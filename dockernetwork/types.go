package dockernetwork

import (
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-sdk/dockerclient"
)

// Network represents a Docker network.
type Network struct {
	response     network.CreateResponse
	inspect      network.Inspect
	dockerClient *dockerclient.Client
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
