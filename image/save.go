package image

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	dockerclient "github.com/docker/docker/client"
	"github.com/docker/go-sdk/client"
)

// ImagePullClient is a client that can pull images.
type ImageSaveClient interface {
	ImageClient

	// ImageSave saves an image to a file.
	ImageSave(ctx context.Context, images []string, saveOptions ...dockerclient.ImageSaveOption) (io.ReadCloser, error)
}

// Save saves an image to a file.
func Save(ctx context.Context, output string, img string, opts ...SaveOption) error {
	saveOpts := &saveOptions{}
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

	if saveOpts.saveClient == nil {
		saveOpts.saveClient = client.DefaultClient
	}

	outputFile, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("open output file %w", err)
	}
	defer func() {
		_ = outputFile.Close()
	}()

	imageReader, err := saveOpts.saveClient.ImageSave(ctx, []string{img}, saveOpts.saveOptions...)
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
