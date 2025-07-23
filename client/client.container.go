package client

import (
	"context"
	"fmt"
	"io"

	"github.com/containerd/errdefs"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
)

// ContainerCreate creates a new container.
func (c *Client) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, name string) (container.CreateResponse, error) {
	dockerClient, err := c.Client()
	if err != nil {
		return container.CreateResponse{}, fmt.Errorf("docker client: %w", err)
	}

	// Add the labels that identify this as a container created by the SDK.
	AddSDKLabels(config.Labels)

	return dockerClient.ContainerCreate(ctx, config, hostConfig, networkingConfig, platform, name)
}

// ContainerExecStart starts a new exec instance.
func (c *Client) ContainerExecAttach(ctx context.Context, execID string, config container.ExecAttachOptions) (types.HijackedResponse, error) {
	dockerClient, err := c.Client()
	if err != nil {
		return types.HijackedResponse{}, fmt.Errorf("docker client: %w", err)
	}

	return dockerClient.ContainerExecAttach(ctx, execID, config)
}

// ContainerExecCreate creates a new exec instance.
func (c *Client) ContainerExecCreate(ctx context.Context, containerID string, options container.ExecOptions) (container.ExecCreateResponse, error) {
	dockerClient, err := c.Client()
	if err != nil {
		return container.ExecCreateResponse{}, fmt.Errorf("docker client: %w", err)
	}

	return dockerClient.ContainerExecCreate(ctx, containerID, options)
}

// ContainerExecInspect inspects a exec instance.
func (c *Client) ContainerExecInspect(ctx context.Context, execID string) (container.ExecInspect, error) {
	dockerClient, err := c.Client()
	if err != nil {
		return container.ExecInspect{}, fmt.Errorf("docker client: %w", err)
	}

	return dockerClient.ContainerExecInspect(ctx, execID)
}

// ContainerInspect inspects a container.
func (c *Client) ContainerInspect(ctx context.Context, containerID string) (container.InspectResponse, error) {
	dockerClient, err := c.Client()
	if err != nil {
		return container.InspectResponse{}, fmt.Errorf("docker client: %w", err)
	}

	return dockerClient.ContainerInspect(ctx, containerID)
}

// ContainerList lists all containers.
func (c *Client) ContainerList(ctx context.Context, options container.ListOptions) ([]container.Summary, error) {
	dockerClient, err := c.Client()
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}

	containers, err := dockerClient.ContainerList(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("container list: %w", err)
	}

	return containers, nil
}

// ContainerLogs returns the logs of a container.
func (c *Client) ContainerLogs(ctx context.Context, containerID string, options container.LogsOptions) (io.ReadCloser, error) {
	dockerClient, err := c.Client()
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}

	return dockerClient.ContainerLogs(ctx, containerID, options)
}

// ContainerPause pauses a container.
func (c *Client) ContainerPause(ctx context.Context, containerID string) error {
	dockerClient, err := c.Client()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}

	if containerID == "" {
		return errdefs.ErrInvalidArgument.WithMessage("containerID is empty")
	}

	return dockerClient.ContainerPause(ctx, containerID)
}

// ContainerUnpause unpauses a container.
func (c *Client) ContainerUnpause(ctx context.Context, containerID string) error {
	dockerClient, err := c.Client()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}

	if containerID == "" {
		return errdefs.ErrInvalidArgument.WithMessage("containerID is empty")
	}

	return dockerClient.ContainerUnpause(ctx, containerID)
}

// ContainerRemove removes a container.
func (c *Client) ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error {
	dockerClient, err := c.Client()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}

	return dockerClient.ContainerRemove(ctx, containerID, options)
}

// ContainerStart starts a container.
func (c *Client) ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error {
	dockerClient, err := c.Client()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}

	return dockerClient.ContainerStart(ctx, containerID, options)
}

// ContainerStop stops a container.
func (c *Client) ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error {
	dockerClient, err := c.Client()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}

	return dockerClient.ContainerStop(ctx, containerID, options)
}

// CopyFromContainer copies a file from a container.
func (c *Client) CopyFromContainer(ctx context.Context, containerID, srcPath string) (io.ReadCloser, container.PathStat, error) {
	dockerClient, err := c.Client()
	if err != nil {
		return nil, container.PathStat{}, fmt.Errorf("docker client: %w", err)
	}

	return dockerClient.CopyFromContainer(ctx, containerID, srcPath)
}

// ContainerLogs returns the logs of a container.
func (c *Client) CopyToContainer(ctx context.Context, containerID, dstPath string, content io.Reader, options container.CopyToContainerOptions) error {
	dockerClient, err := c.Client()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}

	return dockerClient.CopyToContainer(ctx, containerID, dstPath, content, options)
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
