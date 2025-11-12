package client

import (
	"context"
	"fmt"

	"github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

// ContainerCreate creates a new container.
func (c *sdkClient) ContainerCreate(ctx context.Context, options client.ContainerCreateOptions) (client.ContainerCreateResult, error) {
	// Add the labels that identify this as a container created by the SDK.
	AddSDKLabels(options.Config.Labels)

	return c.APIClient.ContainerCreate(ctx, options)
}

// FindContainerByName finds a container by name. The name filter uses a regex to find the containers.
func (c *sdkClient) FindContainerByName(ctx context.Context, name string) (*container.Summary, error) {
	if name == "" {
		return nil, errdefs.ErrInvalidArgument.WithMessage("name is empty")
	}

	// Note that, 'name' filter will use regex to find the containers
	containers, err := c.ContainerList(ctx, client.ContainerListOptions{
		All:     true,
		Filters: make(client.Filters).Add("name", "^"+name+"$"),
	})
	if err != nil {
		return nil, fmt.Errorf("container list: %w", err)
	}

	if len(containers.Items) > 0 {
		return &containers.Items[0], nil
	}

	return nil, errdefs.ErrNotFound.WithMessage(fmt.Sprintf("container %s not found", name))
}
