package volume

import (
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/go-sdk/client"
)

// Volume represents a Docker volume.
type Volume struct {
	*volume.Volume
	dockerClient client.SDKClient
}

// ID is an alias for the Name field, as it coincides with the Name of the volume.
func (v *Volume) ID() string {
	return v.Name
}
