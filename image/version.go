package image

import "github.com/docker/go-sdk/client"

const (
	version     = "0.1.0-alpha010"
	moduleLabel = client.LabelBase + ".image"
)

// Version returns the version of the image package.
func Version() string {
	return version
}
