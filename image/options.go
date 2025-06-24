package image

import (
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/docker/docker/api/types/image"
)

// PullOption is a function that configures the pull options.
type PullOption func(*pullOptions) error

type pullOptions struct {
	pullClient  ImagePullClient
	pullOptions image.PullOptions
}

// WithPullClient sets the pull client used to pull the image.
func WithPullClient(pullClient ImagePullClient) PullOption {
	return func(opts *pullOptions) error {
		opts.pullClient = pullClient
		return nil
	}
}

// WithPullOptions sets the pull options used to pull the image.
func WithPullOptions(imagePullOptions image.PullOptions) PullOption {
	return func(opts *pullOptions) error {
		opts.pullOptions = imagePullOptions
		return nil
	}
}

// SaveOption is a function that configures the save options.
type SaveOption func(*saveOptions) error

type saveOptions struct {
	saveClient ImageSaveClient
	platforms  []ocispec.Platform
}

// WithSaveClient sets the save client used to save the image.
func WithSaveClient(saveClient ImageSaveClient) SaveOption {
	return func(opts *saveOptions) error {
		opts.saveClient = saveClient
		return nil
	}
}

// WithPlatforms sets the platforms to save the image from.
func WithPlatforms(platforms ...ocispec.Platform) SaveOption {
	return func(opts *saveOptions) error {
		opts.platforms = platforms
		return nil
	}
}
