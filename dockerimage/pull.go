package dockerimage

import (
	"context"
	"encoding/base64"
	"encoding/json"
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
func Pull(ctx context.Context, imagePullCli ImagePullClient, imageName string, pullOpt image.PullOptions) error {
	user, pwd, err := dockerconfig.RegistryCredentials(imageName)
	if err != nil {
		imagePullCli.Logger().Warn("failed to get image auth, setting empty credentials for the image", "image", imageName, "error", err)
	} else {
		authConfig := dockerconfig.AuthConfig{
			Username: user,
			Password: pwd,
		}
		encodedJSON, err := json.Marshal(authConfig)
		if err != nil {
			imagePullCli.Logger().Warn("failed to marshal image auth, setting empty credentials for the image", "image", imageName, "error", err)
		} else {
			pullOpt.RegistryAuth = base64.URLEncoding.EncodeToString(encodedJSON)
		}
	}

	var pull io.ReadCloser
	err = backoff.RetryNotify(
		func() error {
			pull, err = imagePullCli.ImagePull(ctx, imageName, pullOpt)
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
			imagePullCli.Logger().Warn("failed to pull image, will retry", "error", err)
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
