package image

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/cenkalti/backoff/v4"
	moby "github.com/docker/docker/client"
	"github.com/docker/go-sdk/client"

	"github.com/docker/go-sdk/config"
	"github.com/docker/go-sdk/config/auth"
)

// defaultPullHandler is the default pull handler function.
// It downloads the entire docker image, and finishes at EOF of the pull request.
// It's up to the caller to handle the io.ReadCloser and close it properly.
var defaultPullHandler = func(r io.ReadCloser) error {
	_, err := io.ReadAll(r)
	return err
}

// SDK should offer utility func to get a DockerClient implementation from docker/cli config or .. any alternatives?
type DockerClient interface {
	API() moby.APIClient
	Config() config.Config
}

// Pull pulls an image from a remote registry, retrying on non-permanent errors.
// See [client.IsPermanentClientError] for the list of non-permanent errors.
// It first extracts the registry credentials from the image name, and sets them in the pull options.
// It needs to be called with a valid image name, and optional pull  options, see [PullOption].
// It's possible to override the default pull handler function by using the [WithPullHandler] option.
func Pull(ctx context.Context, docker DockerClient, imageName string, opts ...PullOption) error {
	pullOpts := &pullOptions{
		pullHandler: defaultPullHandler,
	}
	for _, opt := range opts {
		if err := opt(pullOpts); err != nil {
			return fmt.Errorf("apply pull option: %w", err)
		}
	}

	if imageName == "" {
		return errors.New("image name is not set")
	}

	ref, err := auth.ParseImageRef(imageName)
	if err != nil {
		return fmt.Errorf("parse image ref: %w", err)
	}

	creds, err := docker.Config().AuthConfigForHostname(ref.Registry)
	if err != nil {
		slog.Warn("failed to get registry credentials, setting empty credentials for the image", "image", imageName, "error", err)
	} else {
		authConfig := config.AuthConfig{
			Username: creds.Username,
			Password: creds.Password,
		}
		encodedJSON, err := json.Marshal(authConfig)
		if err != nil {
			slog.Warn("failed to marshal image auth, setting empty credentials for the image", "image", imageName, "error", err)
		} else {
			pullOpts.pullOptions.RegistryAuth = base64.URLEncoding.EncodeToString(encodedJSON)
		}
	}

	var pull io.ReadCloser
	err = backoff.RetryNotify(
		func() error {
			pull, err = docker.API().ImagePull(ctx, imageName, pullOpts.pullOptions)
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
			slog.Warn("failed to pull image, will retry", "error", err)
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
