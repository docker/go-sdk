package volume

import "github.com/docker/go-sdk/client"

const (
	version     = "0.1.0-alpha003"
	moduleLabel = client.LabelBase + ".volume"
)

// Version returns the version of the volume package.
func Version() string {
	return version
}
