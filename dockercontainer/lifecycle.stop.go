package dockercontainer

import "context"

// stoppingHook is a hook that will be called before a container is stopped.
func (c *Container) stoppingHook(ctx context.Context) error {
	return c.applyLifecycleHooks(ctx, false, func(lifecycleHooks LifecycleHooks) error {
		return applyContainerHooks(ctx, lifecycleHooks.PreStops, c)
	})
}

// stoppedHook is a hook that will be called after a container is stopped.
func (c *Container) stoppedHook(ctx context.Context) error {
	return c.applyLifecycleHooks(ctx, false, func(lifecycleHooks LifecycleHooks) error {
		return applyContainerHooks(ctx, lifecycleHooks.PostStops, c)
	})
}
