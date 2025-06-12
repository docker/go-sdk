package dockercontainer

import "context"

// startingHook is a hook that will be called before a container is started.
func (c *Container) startingHook(ctx context.Context) error {
	return c.applyLifecycleHooks(ctx, true, func(lifecycleHooks LifecycleHooks) error {
		return applyContainerHooks(ctx, lifecycleHooks.PreStarts, c)
	})
}

// startedHook is a hook that will be called after a container is started.
func (c *Container) startedHook(ctx context.Context) error {
	return c.applyLifecycleHooks(ctx, true, func(lifecycleHooks LifecycleHooks) error {
		return applyContainerHooks(ctx, lifecycleHooks.PostStarts, c)
	})
}

// readiedHook is a hook that will be called after a container is ready.
func (c *Container) readiedHook(ctx context.Context) error {
	return c.applyLifecycleHooks(ctx, true, func(lifecycleHooks LifecycleHooks) error {
		return applyContainerHooks(ctx, lifecycleHooks.PostReadies, c)
	})
}
