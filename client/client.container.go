package client

import (
	"context"
	"fmt"

	"github.com/containerd/errdefs"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
)

// ContainerCreate creates a new container.
func (c *sdkClient) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, name string) (container.CreateResponse, error) {
	// Add the labels that identify this as a container created by the SDK.
	AddSDKLabels(config.Labels)

	return c.APIClient.ContainerCreate(ctx, config, hostConfig, networkingConfig, platform, name)
}

// FindContainerByName finds a container by name. The name filter uses a regex to find the containers.
func (c *sdkClient) FindContainerByName(ctx context.Context, name string) (*container.Summary, error) {
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
