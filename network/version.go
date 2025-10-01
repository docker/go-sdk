package network

import "github.com/docker/go-sdk/client"

const (
	version     = "0.1.0-alpha010"
	moduleLabel = client.LabelBase + ".network"
)

// Version returns the version of the network package.
func Version() string {
	return version
}
