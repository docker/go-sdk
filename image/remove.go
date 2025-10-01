package image

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/go-sdk/client"
)

// Remove removes an image from the local repository.
func Remove(ctx context.Context, image string, opts ...RemoveOption) ([]image.DeleteResponse, error) {
	removeOpts := &removeOptions{}
	for _, opt := range opts {
		if err := opt(removeOpts); err != nil {
			return nil, fmt.Errorf("apply remove option: %w", err)
		}
	}

	if image == "" {
		return nil, errors.New("image is required")
	}

	if removeOpts.client == nil {
		sdk, err := client.New(ctx)
		if err != nil {
			return nil, err
		}
		removeOpts.client = sdk
	}

	resp, err := removeOpts.client.ImageRemove(ctx, image, removeOpts.removeOptions)
	if err != nil {
		return nil, fmt.Errorf("remove image: %w", err)
	}

	return resp, nil
}
