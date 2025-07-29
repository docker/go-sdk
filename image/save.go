package image

import (
	"context"
	"errors"
	"fmt"
	"os"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	dockerclient "github.com/docker/docker/client"
	"github.com/docker/go-sdk/client"
)

// Save saves an image to a file.
func Save(ctx context.Context, output string, img string, opts ...SaveOption) error {
	saveOpts := &saveOptions{
		platforms: []ocispec.Platform{},
	}
	for _, opt := range opts {
		if err := opt(saveOpts); err != nil {
			return fmt.Errorf("apply save option: %w", err)
		}
	}

	if output == "" {
		return errors.New("output is not set")
	}
	if img == "" {
		return errors.New("image cannot be empty")
	}

	if saveOpts.client == nil {
		sdk, err := client.New(ctx)
		if err != nil {
			return err
		}
		saveOpts.client = sdk
	}

	outputFile, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("open output file %w", err)
	}
	defer func() {
		_ = outputFile.Close()
	}()

	imgSaveOpts := dockerclient.ImageSaveWithPlatforms(saveOpts.platforms...)

	imageReader, err := saveOpts.client.ImageSave(ctx, []string{img}, imgSaveOpts)
	if err != nil {
		return fmt.Errorf("save images %w", err)
	}
	defer func() {
		_ = imageReader.Close()
	}()

	// Attempt optimized readFrom, implemented in linux
	_, err = outputFile.ReadFrom(imageReader)
	if err != nil {
		return fmt.Errorf("write images to output %w", err)
	}

	return nil
}
