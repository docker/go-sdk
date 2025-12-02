package volume

import (
	dockervolume "github.com/moby/moby/api/types/volume"

	"github.com/docker/go-sdk/client"
)

// Volume represents a Docker volume.
type Volume struct {
	*dockervolume.Volume
	dockerClient client.SDKClient
}

// ID is an alias for the Name field, as it coincides with the Name of the volume.
func (v *Volume) ID() string {
	return v.Name
}

// Client returns the client used to create the volume.
func (v *Volume) Client() client.SDKClient {
	return v.dockerClient
}
