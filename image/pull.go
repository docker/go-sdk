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
	"github.com/docker/go-sdk/config/auth"
)

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
func Pull(ctx context.Context, imageName string, opts ...PullOption) error {
	pullOpts := &pullOptions{}
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
		ref, err := auth.ParseImageRef(imageName)
		if err != nil {
			return fmt.Errorf("parse image ref: %w", err)
		}

		creds, ok := authConfigs[ref.Registry]
		if !ok {
			pullOpts.pullClient.Logger().Warn("no image auth found for image, setting empty credentials for the image. This is expected for public images", "image", imageName)
		}

		authConfig := config.AuthConfig{
			Username: creds.Username,
			Password: creds.Password,
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

	// download of docker image finishes at EOF of the pull request
	_, err = io.ReadAll(pull)
	return err
}
