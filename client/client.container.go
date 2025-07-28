package client

import (
	"context"
	"fmt"

	"github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/filters"
	"github.com/moby/moby/api/types/network"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// ContainerCreate creates a new container.
func (c *Client) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, name string) (container.CreateResponse, error) {
	// Add the labels that identify this as a container created by the SDK.
	AddSDKLabels(config.Labels)

	return c.APIClient.ContainerCreate(ctx, config, hostConfig, networkingConfig, platform, name)
}

// FindContainerByName finds a container by name. The name filter uses a regex to find the containers.
func (c *Client) FindContainerByName(ctx context.Context, name string) (*container.Summary, error) {
	if name == "" {
		return nil, errdefs.ErrInvalidArgument.WithMessage("name is empty")
	}

	// Note that, 'name' filter will use regex to find the containers
	filter := filters.NewArgs(filters.Arg("name", fmt.Sprintf("^%s$", name)))
	containers, err := c.ContainerList(ctx, container.ListOptions{All: true, Filters: filter})
	if err != nil {
		return nil, fmt.Errorf("container list: %w", err)
	}

	if len(containers) > 0 {
		return &containers[0], nil
	}

	return nil, errdefs.ErrNotFound.WithMessage(fmt.Sprintf("container %s not found", name))
}
