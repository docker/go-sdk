package volume

import (
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/go-sdk/client"
)

// Volume represents a Docker volume.
type Volume struct {
	*volume.Volume
	dockerClient *client.Client
}
