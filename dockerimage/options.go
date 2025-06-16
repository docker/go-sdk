package dockerimage

import (
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
