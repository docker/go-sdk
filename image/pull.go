package image

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/cenkalti/backoff/v4"

	"github.com/docker/docker/api/types/registry"
	"github.com/docker/go-sdk/client"
)

// defaultPullHandler is the default pull handler function.
// It downloads the entire docker image, and finishes at EOF of the pull request.
// It's up to the caller to handle the io.ReadCloser and close it properly.
var defaultPullHandler = func(r io.ReadCloser) error {
	_, err := io.ReadAll(r)
	return err
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

	if pullOpts.client == nil {
		sdk, err := client.New(ctx)
		if err != nil {
			return err
		}
		pullOpts.client = sdk
	}

	if pullOpts.credentialsFn == nil {
		if err := WithCredentialsFromConfig(pullOpts); err != nil {
			return fmt.Errorf("set credentials for pull option: %w", err)
		}
	}

	if imageName == "" {
		return errors.New("image name is not set")
	}

	username, password, err := pullOpts.credentialsFn(imageName)
	if err != nil {
		return fmt.Errorf("failed to retrieve registry credentials for %s: %w", imageName, err)
	}

	authConfig := registry.AuthConfig{
		Username: username,
		Password: password,
	}

	pullOpts.pullOptions.RegistryAuth, err = registry.EncodeAuthConfig(authConfig)
	if err != nil {
		pullOpts.client.Logger().Warn("failed to encode image auth, setting empty credentials for the image", "image", imageName, "error", err)
	}

	var pull io.ReadCloser
	err = backoff.RetryNotify(
		func() error {
			pull, err = pullOpts.client.ImagePull(ctx, imageName, pullOpts.pullOptions)
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
			pullOpts.client.Logger().Warn("failed to pull image, will retry", "error", err)
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
