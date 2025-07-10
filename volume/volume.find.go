package volume

import (
	"context"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/go-sdk/client"
)

// FindByID finds the volume by ID.
func FindByID(ctx context.Context, volumeID string, opts ...FindOptions) (Volume, error) {
	findOpts := &findOptions{}
	for _, opt := range opts {
		if err := opt(findOpts); err != nil {
			return Volume{}, err
		}
	}

	if findOpts.dockerClient == nil {
		findOpts.dockerClient = client.DefaultClient
	}

	v, err := findOpts.dockerClient.VolumeInspect(ctx, volumeID)
	if err != nil {
		return Volume{}, err
	}

	return Volume{
		Volume:       &v,
		dockerClient: findOpts.dockerClient,
	}, nil
}

// List lists volumes.
func List(ctx context.Context, opts ...FindOptions) ([]Volume, error) {
	findOpts := &findOptions{
		filters: filters.NewArgs(),
	}
	for _, opt := range opts {
		if err := opt(findOpts); err != nil {
			return nil, err
		}
	}

	if findOpts.dockerClient == nil {
		findOpts.dockerClient = client.DefaultClient
	}

	response, err := findOpts.dockerClient.VolumeList(ctx, volume.ListOptions{
		Filters: findOpts.filters,
	})
	if err != nil {
		return nil, err
	}

	volumes := make([]Volume, len(response.Volumes))
	for i, v := range response.Volumes {
		volumes[i] = Volume{
			Volume:       v,
			dockerClient: findOpts.dockerClient,
		}
	}

	for _, w := range response.Warnings {
		findOpts.dockerClient.Logger().Warn(w)
	}

	return volumes, nil
}
