package container

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"strings"
	"time"

	"github.com/containerd/errdefs"
	"github.com/containerd/platforms"

	apiimage "github.com/docker/docker/api/types/image"
	"github.com/docker/go-sdk/container/exec"
	"github.com/docker/go-sdk/container/wait"
	"github.com/docker/go-sdk/image"
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

// Core interface - always available
type ContainerInfo interface {
	ID() string
	Image() string
	ShortID() string
	Logger() *slog.Logger
}

// Optional capability interfaces

// ContainerExecutor is an optional capability interface that can be used to execute commands in the container.
type ContainerExecutor interface {
	Exec(ctx context.Context, cmd []string, opts ...exec.ProcessOption) (int, io.Reader, error)
}

// ContainerFileOperator is an optional capability interface that can be used to copy files to and from the container.
type ContainerFileOperator interface {
	CopyDirToContainer(ctx context.Context, hostDirPath string, containerFilePath string, fileMode int64) error
	CopyToContainer(ctx context.Context, fileContent []byte, containerFilePath string, fileMode int64) error
}

// ContainerWaiter is an optional capability interface that can be used to wait for the container to be ready.
// It embeds the [wait.StrategyTarget] interface to allow the wait strategy to be used to wait for the container to be ready.
type ContainerWaiter interface {
	wait.StrategyTarget
	WaitingFor() wait.Strategy
}

// ContainerStateManager is an optional capability interface that can be used to manage the state of the container.
type ContainerStateManager interface {
	IsRunning() bool
	Running(b bool)
}

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
// It receives a [ContainerInfo] interface, allowing custom implementations
// to be used with the SDK.
type ContainerHook func(ctx context.Context, ctrInfo ContainerInfo) error

// DefaultLoggingHook is a hook that will log the container lifecycle events
var DefaultLoggingHook = LifecycleHooks{
	PreCreates: []DefinitionHook{
		func(_ context.Context, def *Definition) error {
			def.dockerClient.Logger().Info("Creating container", "image", def.image)
			return nil
		},
	},
	PostCreates: []ContainerHook{
		func(_ context.Context, c ContainerInfo) error {
			c.Logger().Info("Container created", "containerID", c.ShortID())
			return nil
		},
	},
	PreStarts: []ContainerHook{
		func(_ context.Context, c ContainerInfo) error {
			c.Logger().Info("Starting container", "containerID", c.ShortID())
			return nil
		},
	},
	PostStarts: []ContainerHook{
		func(_ context.Context, c ContainerInfo) error {
			c.Logger().Info("Container started", "containerID", c.ShortID())
			return nil
		},
	},
	PostReadies: []ContainerHook{
		func(_ context.Context, c ContainerInfo) error {
			c.Logger().Info("Container is ready", "containerID", c.ShortID())
			return nil
		},
	},
	PreStops: []ContainerHook{
		func(_ context.Context, c ContainerInfo) error {
			c.Logger().Info("Stopping container", "containerID", c.ShortID())
			return nil
		},
	},
	PostStops: []ContainerHook{
		func(_ context.Context, c ContainerInfo) error {
			c.Logger().Info("Container stopped", "containerID", c.ShortID())
			return nil
		},
	},
	PreTerminates: []ContainerHook{
		func(_ context.Context, c ContainerInfo) error {
			c.Logger().Info("Terminating container", "containerID", c.ShortID())
			return nil
		},
	},
	PostTerminates: []ContainerHook{
		func(_ context.Context, c ContainerInfo) error {
			c.Logger().Info("Container terminated", "containerID", c.ShortID())
			return nil
		},
	},
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

	// Always append the default pull hook after any other pre-create hook.
	// User could have defined a build hook in which the image has not been defined yet.
	hooks.PreCreates = append(hooks.PreCreates, defaultPullHook...)

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
	for _, hook := range hooks {
		if err := hook(ctx, def); err != nil {
			return fmt.Errorf("apply definition hook: %w", err)
		}
	}

	return nil
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

// defaultPullHook is a hook that will pull the image if it is not present or if the platform is different.
// It must be used as a [DefinitionHook] and not as a [ContainerHook] because it needs to be executed before the container is created.
var defaultPullHook = []DefinitionHook{
	func(ctx context.Context, def *Definition) error {
		var platform *platforms.Platform

		if def.imagePlatform != "" {
			p, err := platforms.Parse(def.imagePlatform)
			if err != nil {
				return fmt.Errorf("invalid platform %s: %w", def.imagePlatform, err)
			}
			platform = &p
			def.platform = platform
		}

		var shouldPullImage bool

		if def.alwaysPullImage {
			shouldPullImage = true // If requested always attempt to pull image
		} else {
			img, err := def.dockerClient.ImageInspect(ctx, def.image)
			if err != nil {
				if !errdefs.IsNotFound(err) {
					return err
				}
				shouldPullImage = true
			}
			if platform != nil && (img.Architecture != platform.Architecture || img.Os != platform.OS) {
				shouldPullImage = true
			}
		}

		if shouldPullImage {
			pullOpt := apiimage.PullOptions{
				Platform: def.imagePlatform, // may be empty
			}
			if err := image.Pull(ctx, def.image, image.WithPullClient(def.dockerClient), image.WithPullOptions(pullOpt)); err != nil {
				return err
			}
		}

		return nil
	},
}
