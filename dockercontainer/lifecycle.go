package dockercontainer

import (
	"context"
	"log/slog"
)

type LifecycleHooks struct {
	PostBuilds     []DefinitionHook
	PreCreates     []DefinitionHook
	PostCreates    []ContainerHook
	PreStarts      []ContainerHook
	PostStarts     []ContainerHook
	PostReadies    []ContainerHook
	PreStops       []ContainerHook
	PostStops      []ContainerHook
	PreTerminates  []ContainerHook
	PostTerminates []ContainerHook
}

// DefinitionHook is a hook that will be called before a container is started.
// It can be used to modify the container definition on container creation,
// using the different lifecycle hooks that are available:
// - Building
// - Creating
// For that, it will receive a Definition, modify it and return an error if needed.
type DefinitionHook func(ctx context.Context, def Definition) error

// ContainerHook is a hook that is called after a container is created
// It can be used to modify the state of the container after it is created,
// using the different lifecycle hooks that are available:
// - Created
// - Starting
// - Started
// - Readied
// - Stopping
// - Stopped
// - Terminating
// - Terminated
// It receives a [Container], modify it and return an error if needed.
type ContainerHook func(ctx context.Context, ctr Container) error

// DefaultLoggingHook is a hook that will log the container lifecycle events
var DefaultLoggingHook = func(logger *slog.Logger) LifecycleHooks {
	shortContainerID := func(c Container) string {
		return c.ID[:12]
	}

	return LifecycleHooks{
		PostBuilds: []DefinitionHook{
			func(_ context.Context, def Definition) error {
				logger.Info("Built image", "image", def.Image)
				return nil
			},
		},
		PreCreates: []DefinitionHook{
			func(_ context.Context, def Definition) error {
				logger.Info("Creating container", "image", def.Image)
				return nil
			},
		},
		PostCreates: []ContainerHook{
			func(_ context.Context, c Container) error {
				logger.Info("Container created", "containerID", shortContainerID(c))
				return nil
			},
		},
		PreStarts: []ContainerHook{
			func(_ context.Context, c Container) error {
				logger.Info("Starting container", "containerID", shortContainerID(c))
				return nil
			},
		},
		PostStarts: []ContainerHook{
			func(_ context.Context, c Container) error {
				logger.Info("Container started", "containerID", shortContainerID(c))
				return nil
			},
		},
		PostReadies: []ContainerHook{
			func(_ context.Context, c Container) error {
				logger.Info("Container is ready", "containerID", shortContainerID(c))
				return nil
			},
		},
		PreStops: []ContainerHook{
			func(_ context.Context, c Container) error {
				logger.Info("Stopping container", "containerID", shortContainerID(c))
				return nil
			},
		},
		PostStops: []ContainerHook{
			func(_ context.Context, c Container) error {
				logger.Info("Container stopped", "containerID", shortContainerID(c))
				return nil
			},
		},
		PreTerminates: []ContainerHook{
			func(_ context.Context, c Container) error {
				logger.Info("Terminating container", "containerID", shortContainerID(c))
				return nil
			},
		},
		PostTerminates: []ContainerHook{
			func(_ context.Context, c Container) error {
				logger.Info("Container terminated", "containerID", shortContainerID(c))
				return nil
			},
		},
	}
}
