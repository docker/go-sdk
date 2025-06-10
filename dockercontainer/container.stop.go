package dockercontainer

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
)

// StopOptions is a type that holds the options for stopping a container.
type StopOptions struct {
	ctx         context.Context
	stopTimeout time.Duration
}

// StopOption is a type that represents an option for stopping a container.
type StopOption func(*StopOptions)

// Context returns the context to use during a Stop or Terminate.
func (o *StopOptions) Context() context.Context {
	return o.ctx
}

// StopTimeout returns the stop timeout to use during a Stop or Terminate.
func (o *StopOptions) StopTimeout() time.Duration {
	return o.stopTimeout
}

// StopTimeout returns a StopOption that sets the timeout.
// Default: See [Container.Stop].
func StopTimeout(timeout time.Duration) StopOption {
	return func(c *StopOptions) {
		c.stopTimeout = timeout
	}
}

// NewStopOptions returns a fully initialised StopOptions.
// Defaults: StopTimeout: 10 seconds.
func NewStopOptions(ctx context.Context, opts ...StopOption) *StopOptions {
	options := &StopOptions{
		stopTimeout: time.Second * 10,
		ctx:         ctx,
	}
	for _, opt := range opts {
		opt(options)
	}
	return options
}

// Stop stops the container.
//
// In case the container fails to stop gracefully within a time frame specified
// by the timeout argument, it is forcefully terminated (killed).
//
// If no timeout is passed, the default StopTimeout value is used, 10 seconds,
// otherwise the engine default. A negative timeout value can be specified,
// meaning no timeout, i.e. no forceful termination is performed.
//
// All hooks are called in the following order:
//   - [LifecycleHooks.PreStops]
//   - [LifecycleHooks.PostStops]
//
// If the container is already stopped, the method is a no-op.
func (c *Container) Stop(ctx context.Context, opts ...StopOption) error {
	stopOptions := NewStopOptions(ctx, opts...)

	err := c.stoppingHook(stopOptions.Context())
	if err != nil {
		return fmt.Errorf("stopping hook: %w", err)
	}

	var options container.StopOptions

	timeoutSeconds := int(stopOptions.StopTimeout().Seconds())
	options.Timeout = &timeoutSeconds

	if err := c.dockerClient.ContainerStop(stopOptions.Context(), c.ID, options); err != nil {
		return fmt.Errorf("container stop: %w", err)
	}

	c.isRunning = false

	err = c.stoppedHook(stopOptions.Context())
	if err != nil {
		return fmt.Errorf("stopped hook: %w", err)
	}

	return nil
}
