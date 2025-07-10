package volume

import (
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/go-sdk/client"
)

type options struct {
	client *client.Client
	labels map[string]string
	name   string
}

// Option is a function that modifies the options to create a volume.
type Option func(*options) error

// WithClient sets the docker client.
func WithClient(client *client.Client) Option {
	return func(o *options) error {
		o.client = client
		return nil
	}
}

// WithName sets the name of the volume.
func WithName(name string) Option {
	return func(o *options) error {
		o.name = name
		return nil
	}
}

// WithLabels sets the labels of the volume.
func WithLabels(labels map[string]string) Option {
	return func(o *options) error {
		o.labels = labels
		return nil
	}
}

type TerminateOption func(*terminateOptions) error

type terminateOptions struct {
	force bool
}

// WithForce sets the force option.
func WithForce() TerminateOption {
	return func(o *terminateOptions) error {
		o.force = true
		return nil
	}
}

type findOptions struct {
	dockerClient *client.Client
	filters      filters.Args
}

// FindOptions is a function that modifies the find options
// used to find volumes.
type FindOptions func(opts *findOptions) error

// WithFindClient returns an [FindOptions] that sets the find client.
func WithFindClient(dockerClient *client.Client) FindOptions {
	return func(o *findOptions) error {
		o.dockerClient = dockerClient
		return nil
	}
}

// WithFilters sets the filters to be used to filter the volumes.
func WithFilters(filters filters.Args) FindOptions {
	return func(opts *findOptions) error {
		opts.filters = filters
		return nil
	}
}
