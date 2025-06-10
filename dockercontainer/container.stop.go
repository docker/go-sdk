package dockercontainer

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
)

// Stop stops the container.
//
// In case the container fails to stop gracefully within a time frame specified
// by the timeout argument, it is forcefully terminated (killed).
//
// If the timeout is nil, the container's StopTimeout value is used, if set,
// otherwise the engine default. A negative timeout value can be specified,
// meaning no timeout, i.e. no forceful termination is performed.
//
// All hooks are called in the following order:
//   - [LifecycleHooks.PreStops]
//   - [LifecycleHooks.PostStops]
//
// If the container is already stopped, the method is a no-op.
func (c *Container) Stop(ctx context.Context, timeout *time.Duration) error {
	// Note we can't check isRunning here because we allow external creation
	// without exposing the ability to fully initialize the container state.
	// See: https://github.com/testcontainers/testcontainers-go/issues/2667
	// TODO: Add a check for isRunning when the above issue is resolved.
	err := c.stoppingHook(ctx)
	if err != nil {
		return fmt.Errorf("stopping hook: %w", err)
	}

	var options container.StopOptions

	if timeout != nil {
		timeoutSeconds := int(timeout.Seconds())
		options.Timeout = &timeoutSeconds
	}

	if err := c.dockerClient.ContainerStop(ctx, c.ID, options); err != nil {
		return fmt.Errorf("container stop: %w", err)
	}

	c.isRunning = false

	err = c.stoppedHook(ctx)
	if err != nil {
		return fmt.Errorf("stopped hook: %w", err)
	}

	return nil
}
