package image

import (
	"context"
	"errors"
	"fmt"

	dockerclient "github.com/moby/moby/client"

	"github.com/docker/go-sdk/client"
)

// Remove removes an image from the local repository.
func Remove(ctx context.Context, image string, opts ...RemoveOption) (dockerclient.ImageRemoveResult, error) {
	removeOpts := &removeOptions{}
	for _, opt := range opts {
		if err := opt(removeOpts); err != nil {
			return dockerclient.ImageRemoveResult{}, fmt.Errorf("apply remove option: %w", err)
		}
	}

	if image == "" {
		return dockerclient.ImageRemoveResult{}, errors.New("image is required")
	}

	if removeOpts.client == nil {
		sdk, err := client.New(ctx)
		if err != nil {
			return dockerclient.ImageRemoveResult{}, err
		}
		removeOpts.client = sdk
	}

	resp, err := removeOpts.client.ImageRemove(ctx, image, removeOpts.removeOptions)
	if err != nil {
		return dockerclient.ImageRemoveResult{}, fmt.Errorf("remove image: %w", err)
	}

	return resp, nil
}
