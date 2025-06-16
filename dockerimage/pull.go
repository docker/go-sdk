package dockerimage

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
	"github.com/docker/go-sdk/dockerclient"
	"github.com/docker/go-sdk/dockerconfig"
)

// ImagePullClient is a client that can pull images.
type ImagePullClient interface {
	ImageClient

	// ImagePull pulls an image from a remote registry.
	ImagePull(ctx context.Context, image string, options image.PullOptions) (io.ReadCloser, error)
}

// Pull pulls an image from a remote registry, retrying on non-permanent errors.
// See [dockerclient.IsPermanentClientError] for the list of non-permanent errors.
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
		// create a new docker client if not set
		cli, err := dockerclient.New(ctx)
		if err != nil {
			return fmt.Errorf("create docker client: %w", err)
		}
		pullOpts.pullClient = cli
		// In case there is no pull client set, we use the default docker client
		// to pull the image. We need to close it when done.
		defer cli.Close()
	}

	if imageName == "" {
		return errors.New("image name is not set")
	}

	creds, err := dockerconfig.RegistryCredentials(imageName)
	if err != nil {
		pullOpts.pullClient.Logger().Warn("failed to get image auth, setting empty credentials for the image", "image", imageName, "error", err)
	} else {
		authConfig := dockerconfig.AuthConfig{
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
				if dockerclient.IsPermanentClientError(err) {
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
