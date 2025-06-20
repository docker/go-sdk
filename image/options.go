package image

import (
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
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
	saveClient  ImageSaveClient
	saveOptions []client.ImageSaveOption
}

// WithSaveClient sets the save client used to save the image.
func WithSaveClient(saveClient ImageSaveClient) SaveOption {
	return func(opts *saveOptions) error {
		opts.saveClient = saveClient
		return nil
	}
}

// WithSaveOptions sets the save options used to save the image.
func WithSaveOptions(options ...client.ImageSaveOption) SaveOption {
	return func(opts *saveOptions) error {
		opts.saveOptions = options
		return nil
	}
}
