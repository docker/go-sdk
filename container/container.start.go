package container

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
)

// Start will start an already created container
func (c *Container) Start(ctx context.Context) error {
	err := c.startingHook(ctx)
	if err != nil {
		return fmt.Errorf("starting hook: %w", err)
	}

	if err := c.dockerClient.ContainerStart(ctx, c.ID(), container.StartOptions{}); err != nil {
		return fmt.Errorf("container start: %w", err)
	}
	defer c.dockerClient.Close()

	err = c.startedHook(ctx)
	if err != nil {
		return fmt.Errorf("started hook: %w", err)
	}

	c.isRunning = true

	err = c.readiedHook(ctx)
	if err != nil {
		return fmt.Errorf("readied hook: %w", err)
	}

	return nil
}
