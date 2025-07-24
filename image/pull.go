package image

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/cenkalti/backoff/v4"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/go-sdk/client"
	"github.com/docker/go-sdk/config"
)

// defaultPullHandler is the default pull handler function.
// It downloads the entire docker image, and finishes at EOF of the pull request.
// It's up to the caller to handle the io.ReadCloser and close it properly.
var defaultPullHandler = func(r io.ReadCloser) error {
	_, err := io.ReadAll(r)
	return err
}

// ImagePullClient is a client that can pull images.
type ImagePullClient interface {
	ImageClient

	// ImagePull pulls an image from a remote registry.
	ImagePull(ctx context.Context, image string, options image.PullOptions) (io.ReadCloser, error)
}

// Pull pulls an image from a remote registry, retrying on non-permanent errors.
// See [client.IsPermanentClientError] for the list of non-permanent errors.
// It first extracts the registry credentials from the image name, and sets them in the pull options.
// It needs to be called with a valid image name, and optional pull  options, see [PullOption].
// It's possible to override the default pull handler function by using the [WithPullHandler] option.
func Pull(ctx context.Context, imageName string, opts ...PullOption) error {
	pullOpts := &pullOptions{
		pullHandler: defaultPullHandler,
	}
	for _, opt := range opts {
		if err := opt(pullOpts); err != nil {
			return fmt.Errorf("apply pull option: %w", err)
		}
	}

	if pullOpts.pullClient == nil {
		pullOpts.pullClient = client.DefaultClient
		// In case there is no pull client set, we use the default docker client
		// to pull the image. We need to close it when done.
		defer pullOpts.pullClient.Close()
	}

	if imageName == "" {
		return errors.New("image name is not set")
	}

	authConfigs, err := config.AuthConfigs(imageName)
	if err != nil {
		pullOpts.pullClient.Logger().Warn("failed to get image auth, setting empty credentials for the image", "image", imageName, "error", err)
	} else {
		// there must be only one auth config for the image
		if len(authConfigs) > 1 {
			return fmt.Errorf("multiple auth configs found for image %s, expected only one", imageName)
		}

		var tmp config.AuthConfig
		for _, ac := range authConfigs {
			tmp = ac
		}

		authConfig := config.AuthConfig{
			Username: tmp.Username,
			Password: tmp.Password,
		}
		encodedJSON, err := json.Marshal(authConfig)
		if err != nil {
			pullOpts.pullClient.Logger().Warn("failed to marshal image auth, setting empty credentials for the image", "image", imageName, "error", err)
		} else {
			pullOpts.pullOptions.RegistryAuth = base64.URLEncoding.EncodeToString(encodedJSON)
		}
	}

	var pull io.ReadCloser
	err = backoff.RetryNotify(
		func() error {
			pull, err = pullOpts.pullClient.ImagePull(ctx, imageName, pullOpts.pullOptions)
			if err != nil {
				if client.IsPermanentClientError(err) {
					return backoff.Permanent(err)
				}
				return err
			}

			return nil
		},
		backoff.WithContext(backoff.NewExponentialBackOff(), ctx),
		func(err error, _ time.Duration) {
			pullOpts.pullClient.Logger().Warn("failed to pull image, will retry", "error", err)
		},
	)
	if err != nil {
		return err
	}
	defer pull.Close()

	if err := pullOpts.pullHandler(pull); err != nil {
		return fmt.Errorf("pull handler: %w", err)
	}

	return err
}
