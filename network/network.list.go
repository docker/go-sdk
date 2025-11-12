package network

import (
	"context"
	"errors"
	"fmt"
	"maps"

	"github.com/moby/moby/api/types/network"
	dockerclient "github.com/moby/moby/client"

	"github.com/docker/go-sdk/client"
)

const (
	// filterByID uses to filter network by identifier.
	filterByID = "id"

	// filterByName uses to filter network by name.
	filterByName = "name"
)

type listOptions struct {
	client  client.SDKClient
	filters dockerclient.Filters
}

type ListOptions func(opts *listOptions) error

// WithListClient sets the client to be used to list the networks.
func WithListClient(client client.SDKClient) ListOptions {
	return func(opts *listOptions) error {
		opts.client = client
		return nil
	}
}

// WithFilters sets the filters to be used to filter the networks.
func WithFilters(filters dockerclient.Filters) ListOptions {
	return func(opts *listOptions) error {
		opts.filters = maps.Clone(filters)
		return nil
	}
}

// FindByID returns a network by its ID.
func FindByID(ctx context.Context, id string, opts ...ListOptions) (network.Summary, error) {
	opts = append(opts, WithFilters(make(dockerclient.Filters).Add(filterByID, id)))

	nws, err := list(ctx, opts...)
	if err != nil {
		return network.Summary{}, err
	}

	return nws[0], nil
}

// FindByName returns a network by its name.
func FindByName(ctx context.Context, name string, opts ...ListOptions) (network.Summary, error) {
	opts = append(opts, WithFilters(make(dockerclient.Filters).Add(filterByName, name)))

	nws, err := list(ctx, opts...)
	if err != nil {
		return network.Summary{}, err
	}

	return nws[0], nil
}

// List returns a list of networks.
func List(ctx context.Context, opts ...ListOptions) ([]network.Summary, error) {
	return list(ctx, opts...)
}

func list(ctx context.Context, opts ...ListOptions) ([]network.Summary, error) {
	var nws []network.Summary // initialize to the zero value

	initialOpts := &listOptions{
		filters: make(dockerclient.Filters),
	}
	for _, opt := range opts {
		if err := opt(initialOpts); err != nil {
			return nws, err
		}
	}

	nwOpts := dockerclient.NetworkListOptions{}
	if len(initialOpts.filters) > 0 {
		nwOpts.Filters = initialOpts.filters
	}

	if initialOpts.client == nil {
		sdk, err := client.New(ctx)
		if err != nil {
			return nil, err
		}
		initialOpts.client = sdk
	}

	list, err := initialOpts.client.NetworkList(ctx, nwOpts)
	if err != nil {
		return nws, fmt.Errorf("failed to list networks: %w", err)
	}

	if len(list.Items) == 0 {
		return nws, errors.New("no networks found")
	}

	nws = append(nws, list.Items...)

	return nws, nil
}
