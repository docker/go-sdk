package image

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/moby/term"

	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/go-sdk/client"
)

// DisplayProgress creates a pull handler that displays formatted pull progress to the given writer.
// It automatically detects if the writer is a terminal and enables colors and progress bars accordingly.
// If the writer is not a terminal, it displays plain text progress information.
//
// This is useful when you want to customize where progress is displayed while maintaining
// the formatted output. For example:
//
//	err := image.Pull(ctx, "nginx:latest",
//		image.WithPullHandler(image.DisplayProgress(os.Stderr)))
func DisplayProgress(out io.Writer) func(r io.ReadCloser) error {
	return func(r io.ReadCloser) error {
		// Get file descriptor and check if it's a terminal
		fd, isTerm := term.GetFdInfo(out)

		// Use Docker's jsonmessage package to properly display pull progress
		// This will render progress bars on terminals or plain text otherwise
		return jsonmessage.DisplayJSONMessagesStream(r, out, fd, isTerm, nil)
	}
}

// defaultPullHandler is the default pull handler function.
// It downloads the entire docker image, and finishes at EOF of the pull request.
// It properly formats JSON progress messages for terminal display using Docker's jsonmessage package.
// It's up to the caller to handle the io.ReadCloser and close it properly.
var defaultPullHandler = DisplayProgress(os.Stdout)

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
