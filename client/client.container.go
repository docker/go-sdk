package client

import (
	"context"
	"io"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
)

// ContainerCreate creates a new container.
func (c *Client) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, name string) (container.CreateResponse, error) {
	return c.Client().ContainerCreate(ctx, config, hostConfig, networkingConfig, platform, name)
}

// ContainerExecStart starts a new exec instance.
func (c *Client) ContainerExecAttach(ctx context.Context, execID string, config container.ExecAttachOptions) (types.HijackedResponse, error) {
	return c.Client().ContainerExecAttach(ctx, execID, config)
}

// ContainerExecCreate creates a new exec instance.
func (c *Client) ContainerExecCreate(ctx context.Context, container string, options container.ExecOptions) (container.ExecCreateResponse, error) {
	return c.Client().ContainerExecCreate(ctx, container, options)
}

// ContainerExecInspect inspects a exec instance.
func (c *Client) ContainerExecInspect(ctx context.Context, execID string) (container.ExecInspect, error) {
	return c.Client().ContainerExecInspect(ctx, execID)
}

// ContainerInspect inspects a container.
func (c *Client) ContainerInspect(ctx context.Context, containerID string) (container.InspectResponse, error) {
	return c.Client().ContainerInspect(ctx, containerID)
}

// ContainerLogs returns the logs of a container.
func (c *Client) ContainerLogs(ctx context.Context, containerID string, options container.LogsOptions) (io.ReadCloser, error) {
	return c.Client().ContainerLogs(ctx, containerID, options)
}

// ContainerRemove removes a container.
func (c *Client) ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error {
	return c.Client().ContainerRemove(ctx, containerID, options)
}

// ContainerStart starts a container.
func (c *Client) ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error {
	return c.Client().ContainerStart(ctx, containerID, options)
}

// ContainerStop stops a container.
func (c *Client) ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error {
	return c.Client().ContainerStop(ctx, containerID, options)
}

// CopyFromContainer copies a file from a container.
func (c *Client) CopyFromContainer(ctx context.Context, containerID, srcPath string) (io.ReadCloser, container.PathStat, error) {
	return c.Client().CopyFromContainer(ctx, containerID, srcPath)
}

// ContainerLogs returns the logs of a container.
func (c *Client) CopyToContainer(ctx context.Context, containerID, dstPath string, content io.Reader, options container.CopyToContainerOptions) error {
	return c.Client().CopyToContainer(ctx, containerID, dstPath, content, options)
}
