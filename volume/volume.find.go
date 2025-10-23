package volume

import (
	"context"

	dockerclient "github.com/moby/moby/client"

	"github.com/docker/go-sdk/client"
)

// FindByID finds the volume by ID.
func FindByID(ctx context.Context, volumeID string, opts ...FindOptions) (*Volume, error) {
	findOpts := &findOptions{}
	for _, opt := range opts {
		if err := opt(findOpts); err != nil {
			return nil, err
		}
	}

	if findOpts.client == nil {
		sdk, err := client.New(ctx)
		if err != nil {
			return nil, err
		}
		findOpts.client = sdk
	}

	v, err := findOpts.client.VolumeInspect(ctx, volumeID, dockerclient.VolumeInspectOptions{})
	if err != nil {
		return nil, err
	}

	return &Volume{
		Volume:       &v.Volume,
		dockerClient: findOpts.client,
	}, nil
}

// List lists volumes.
func List(ctx context.Context, opts ...FindOptions) ([]Volume, error) {
	findOpts := &findOptions{
		filters: dockerclient.Filters{},
	}
	for _, opt := range opts {
		if err := opt(findOpts); err != nil {
			return nil, err
		}
	}

	if findOpts.client == nil {
		sdk, err := client.New(ctx)
		if err != nil {
			return nil, err
		}
		findOpts.client = sdk
	}

	response, err := findOpts.client.VolumeList(ctx, dockerclient.VolumeListOptions{
		Filters: findOpts.filters,
	})
	if err != nil {
		return nil, err
	}

	volumes := make([]Volume, len(response.Items))
	for i, v := range response.Items {
		volumes[i] = Volume{
			Volume:       &v,
			dockerClient: findOpts.client,
		}
	}

	for _, w := range response.Warnings {
		findOpts.client.Logger().Warn(w)
	}

	return volumes, nil
}
