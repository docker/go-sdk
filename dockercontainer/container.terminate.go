package dockercontainer

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-sdk/dockerclient"
)

// TerminableContainer is a container that can be terminated.
type TerminableContainer interface {
	Terminate(ctx context.Context, opts ...TerminateOption) error
}

// TerminateOptions is a type that holds the options for terminating a container.
type TerminateOptions struct {
	*StopOptions
	volumes []string
}

// TerminateOption is a type that represents an option for terminating a container.
type TerminateOption func(*TerminateOptions)

// NewTerminateOptions returns a fully initialised TerminateOptions.
// Defaults: StopTimeout: 10 seconds.
func NewTerminateOptions(ctx context.Context, opts ...TerminateOption) *TerminateOptions {
	options := &TerminateOptions{
		StopOptions: NewStopOptions(ctx),
	}
	for _, opt := range opts {
		opt(options)
	}
	return options
}

// Cleanup performs any clean up needed
func (o *TerminateOptions) Cleanup(cli *dockerclient.Client) error {
	if len(o.volumes) == 0 {
		return nil
	}

	// Best effort to remove all volumes.
	var errs []error
	for _, volume := range o.volumes {
		if errRemove := cli.VolumeRemove(o.ctx, volume, true); errRemove != nil {
			errs = append(errs, fmt.Errorf("volume remove %q: %w", volume, errRemove))
		}
	}
	return errors.Join(errs...)
}

// TerminateTimeout returns a TerminateOption that sets the timeout.
// Default: See [Container.Stop].
func TerminateTimeout(timeout time.Duration) TerminateOption {
	return func(c *TerminateOptions) {
		c.stopTimeout = timeout
	}
}

// RemoveVolumes returns a TerminateOption that sets additional volumes to remove.
// This is useful when the container creates named volumes that should be removed
// which are not removed by default.
// Default: nil.
func RemoveVolumes(volumes ...string) TerminateOption {
	return func(c *TerminateOptions) {
		c.volumes = volumes
	}
}

// TerminateContainer calls [TerminableContainer.Terminate] on the container if it is not nil.
//
// This should be called as a defer directly after [Create](...)
// to ensure the container is terminated when the function ends.
func TerminateContainer(ctr TerminableContainer, options ...TerminateOption) error {
	if isNil(ctr) {
		return nil
	}

	err := ctr.Terminate(context.Background(), options...)
	if !isCleanupSafe(err) {
		return fmt.Errorf("terminate: %w", err)
	}

	return nil
}

// isNil returns true if val is nil or a nil instance false otherwise.
func isNil(val any) bool {
	if val == nil {
		return true
	}

	valueOf := reflect.ValueOf(val)
	switch valueOf.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return valueOf.IsNil()
	default:
		return false
	}
}

// Terminate calls stops and then removes the container including its volumes.
// If its image was built it and all child images are also removed unless
// the [FromDockerfile.KeepImage] on the [ContainerRequest] was set to true.
//
// The following hooks are called in order:
//   - [LifecycleHooks.PreTerminates]
//   - [LifecycleHooks.PostTerminates]
//
// Default: timeout is 10 seconds.
func (c *Container) Terminate(ctx context.Context, opts ...TerminateOption) error {
	options := NewTerminateOptions(ctx, opts...)
	err := c.Stop(options.Context(), StopTimeout(options.StopTimeout()))
	if err != nil && !isCleanupSafe(err) {
		return fmt.Errorf("stop: %w", err)
	}

	// TODO: Handle errors from ContainerRemove more correctly, e.g. should we
	// run the terminated hook?
	errs := []error{
		c.terminatingHook(ctx),
		c.dockerClient.ContainerRemove(ctx, c.ID(), container.RemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		}),
		c.terminatedHook(ctx),
	}

	c.isRunning = false

	if err = options.Cleanup(c.dockerClient); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}
