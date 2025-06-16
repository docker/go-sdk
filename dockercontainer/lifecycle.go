package dockercontainer

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"strings"
	"time"
)

type LifecycleHooks struct {
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
type DefinitionHook func(ctx context.Context, def *Definition) error

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
type ContainerHook func(ctx context.Context, ctr *Container) error

// DefaultLoggingHook is a hook that will log the container lifecycle events
var DefaultLoggingHook = func(logger *slog.Logger) LifecycleHooks {
	return LifecycleHooks{
		PreCreates: []DefinitionHook{
			func(_ context.Context, def *Definition) error {
				logger.Info("Creating container", "image", def.image)
				return nil
			},
		},
		PostCreates: []ContainerHook{
			func(_ context.Context, c *Container) error {
				logger.Info("Container created", "containerID", c.shortID)
				return nil
			},
		},
		PreStarts: []ContainerHook{
			func(_ context.Context, c *Container) error {
				logger.Info("Starting container", "containerID", c.shortID)
				return nil
			},
		},
		PostStarts: []ContainerHook{
			func(_ context.Context, c *Container) error {
				logger.Info("Container started", "containerID", c.shortID)
				return nil
			},
		},
		PostReadies: []ContainerHook{
			func(_ context.Context, c *Container) error {
				logger.Info("Container is ready", "containerID", c.shortID)
				return nil
			},
		},
		PreStops: []ContainerHook{
			func(_ context.Context, c *Container) error {
				logger.Info("Stopping container", "containerID", c.shortID)
				return nil
			},
		},
		PostStops: []ContainerHook{
			func(_ context.Context, c *Container) error {
				logger.Info("Container stopped", "containerID", c.shortID)
				return nil
			},
		},
		PreTerminates: []ContainerHook{
			func(_ context.Context, c *Container) error {
				logger.Info("Terminating container", "containerID", c.shortID)
				return nil
			},
		},
		PostTerminates: []ContainerHook{
			func(_ context.Context, c *Container) error {
				logger.Info("Container terminated", "containerID", c.shortID)
				return nil
			},
		},
	}
}

// combineContainerHooks returns a [LifecycleHook] as the result
// of combining the default hooks with the user-defined hooks.
//
// The order of hooks is the following:
// - Pre-hooks run the default hooks first then the user-defined hooks
// - Post-hooks run the user-defined hooks first then the default hooks
// The order of execution will be:
// - default pre-hooks
// - user-defined pre-hooks
// - user-defined post-hooks
// - default post-hooks
func combineContainerHooks(defaultHooks, userDefinedHooks []LifecycleHooks) LifecycleHooks {
	// We use reflection here to ensure that any new hooks are handled.
	var hooks LifecycleHooks
	hooksVal := reflect.ValueOf(&hooks).Elem()
	hooksType := reflect.TypeOf(hooks)
	for _, defaultHook := range defaultHooks {
		defaultVal := reflect.ValueOf(defaultHook)
		for i := range hooksType.NumField() {
			if strings.HasPrefix(hooksType.Field(i).Name, "Pre") {
				field := hooksVal.Field(i)
				field.Set(reflect.AppendSlice(field, defaultVal.Field(i)))
			}
		}
	}

	// Append the user-defined hooks after the default pre-hooks
	// and because the post hooks are still empty, the user-defined
	// post-hooks will be the first ones to be executed.
	for _, userDefinedHook := range userDefinedHooks {
		userVal := reflect.ValueOf(userDefinedHook)
		for i := range hooksType.NumField() {
			field := hooksVal.Field(i)
			field.Set(reflect.AppendSlice(field, userVal.Field(i)))
		}
	}

	// Finally, append the default post-hooks.
	for _, defaultHook := range defaultHooks {
		defaultVal := reflect.ValueOf(defaultHook)
		for i := range hooksType.NumField() {
			if strings.HasPrefix(hooksType.Field(i).Name, "Post") {
				field := hooksVal.Field(i)
				field.Set(reflect.AppendSlice(field, defaultVal.Field(i)))
			}
		}
	}

	return hooks
}

func applyContainerHooks(ctx context.Context, hooks []ContainerHook, ctr *Container) error {
	var errs []error
	for _, hook := range hooks {
		if err := hook(ctx, ctr); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func applyDefinitionHooks(ctx context.Context, hooks []DefinitionHook, def *Definition) error {
	var errs []error
	for _, hook := range hooks {
		if err := hook(ctx, def); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// applyLifecycleHooks calls hook on all LifecycleHooks.
func (def *Definition) applyLifecycleHooks(hook func(lifecycleHooks LifecycleHooks) error) error {
	if def.lifecycleHooks == nil {
		return nil
	}

	var errs []error
	for _, lifecycleHooks := range def.lifecycleHooks {
		if err := hook(lifecycleHooks); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// applyLifecycleHooks applies all lifecycle hooks reporting the container logs on error if logError is true.
func (c *Container) applyLifecycleHooks(ctx context.Context, logError bool, hook func(lifecycleHooks LifecycleHooks) error) error {
	if c.lifecycleHooks == nil {
		return nil
	}

	var errs []error
	for _, lifecycleHooks := range c.lifecycleHooks {
		if err := hook(lifecycleHooks); err != nil {
			errs = append(errs, err)
		}
	}

	if err := errors.Join(errs...); err != nil {
		if logError {
			select {
			case <-ctx.Done():
				// Context has timed out so need a new context to get logs.
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
				defer cancel()
				c.printLogs(ctx, err)
			default:
				c.printLogs(ctx, err)
			}
		}

		return err
	}

	return nil
}
