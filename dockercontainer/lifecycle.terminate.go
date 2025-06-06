package dockercontainer

import "context"

// terminatingHook is a hook that will be called before a container is terminated.
func (c *Container) terminatingHook(ctx context.Context) error {
	return c.applyLifecycleHooks(ctx, false, func(lifecycleHooks LifecycleHooks) error {
		return applyContainerHooks(ctx, lifecycleHooks.PreTerminates, c)
	})
}

// stoppedHook is a hook that will be called after a container is stopped.
func (c *Container) terminatedHook(ctx context.Context) error {
	return c.applyLifecycleHooks(ctx, false, func(lifecycleHooks LifecycleHooks) error {
		return applyContainerHooks(ctx, lifecycleHooks.PostTerminates, c)
	})
}
