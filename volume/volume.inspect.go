package volume

import (
	"context"
	"errors"

	"github.com/moby/moby/client"
)

func (v *Volume) Inspect(ctx context.Context, options client.VolumeInspectOptions) (client.VolumeInspectResult, error) {
	if v.dockerClient == nil {
		return client.VolumeInspectResult{}, errors.New("docker client is not initialized")
	}

	inspect, err := v.dockerClient.VolumeInspect(ctx, v.ID(), options)
	if err != nil {
		return client.VolumeInspectResult{}, err
	}

	return inspect, nil
}
