package container

import "github.com/docker/go-sdk/client"

const (
	version     = "0.1.0-alpha010"
	moduleLabel = client.LabelBase + ".container"
)

// Version returns the version of the container package.
func Version() string {
	return version
}
