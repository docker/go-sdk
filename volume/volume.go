package volume

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/volume"
	"github.com/docker/go-sdk/client"
)

// New creates a new volume.
// If no name is provided, a random name is generated.
// If no client is provided, the default client is used.
func New(ctx context.Context, opts ...Option) (*Volume, error) {
	volumeOptions := &options{
		labels: make(map[string]string),
	}

	for _, opt := range opts {
		if err := opt(volumeOptions); err != nil {
			return nil, fmt.Errorf("apply option: %w", err)
		}
	}

	if volumeOptions.client == nil {
		volumeOptions.client = client.DefaultClient
	}

	volumeOptions.labels[moduleLabel] = Version()

	v, err := volumeOptions.client.VolumeCreate(ctx, volume.CreateOptions{
		Name:   volumeOptions.name,
		Labels: volumeOptions.labels,
	})
	if err != nil {
		return nil, fmt.Errorf("create volume: %w", err)
	}

	return &Volume{
		Volume:       &v,
		dockerClient: volumeOptions.client,
	}, nil
}
