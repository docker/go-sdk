package dockercontainer

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCombineLifecycleHooks(t *testing.T) {
	prints := []string{}

	preCreateFunc := func(prefix string, hook string, lifecycleID int, hookID int) func(_ context.Context, _ *Definition) error {
		return func(_ context.Context, _ *Definition) error {
			prints = append(prints, fmt.Sprintf("[%s] pre-%s hook %d.%d", prefix, hook, lifecycleID, hookID))
			return nil
		}
	}
	hookFunc := func(prefix string, hookType string, hook string, lifecycleID int, hookID int) func(_ context.Context, _ *Container) error {
		return func(_ context.Context, _ *Container) error {
			prints = append(prints, fmt.Sprintf("[%s] %s-%s hook %d.%d", prefix, hookType, hook, lifecycleID, hookID))
			return nil
		}
	}
	preFunc := func(prefix string, hook string, lifecycleID int, hookID int) func(_ context.Context, _ *Container) error {
		return hookFunc(prefix, "pre", hook, lifecycleID, hookID)
	}
	postFunc := func(prefix string, hook string, lifecycleID int, hookID int) func(_ context.Context, _ *Container) error {
		return hookFunc(prefix, "post", hook, lifecycleID, hookID)
	}

	lifecycleHookFunc := func(prefix string, lifecycleID int) LifecycleHooks {
		return LifecycleHooks{
			PreCreates:     []DefinitionHook{preCreateFunc(prefix, "create", lifecycleID, 1), preCreateFunc(prefix, "create", lifecycleID, 2)},
			PostCreates:    []ContainerHook{postFunc(prefix, "create", lifecycleID, 1), postFunc(prefix, "create", lifecycleID, 2)},
			PreStarts:      []ContainerHook{preFunc(prefix, "start", lifecycleID, 1), preFunc(prefix, "start", lifecycleID, 2)},
			PostStarts:     []ContainerHook{postFunc(prefix, "start", lifecycleID, 1), postFunc(prefix, "start", lifecycleID, 2)},
			PostReadies:    []ContainerHook{postFunc(prefix, "ready", lifecycleID, 1), postFunc(prefix, "ready", lifecycleID, 2)},
			PreStops:       []ContainerHook{preFunc(prefix, "stop", lifecycleID, 1), preFunc(prefix, "stop", lifecycleID, 2)},
			PostStops:      []ContainerHook{postFunc(prefix, "stop", lifecycleID, 1), postFunc(prefix, "stop", lifecycleID, 2)},
			PreTerminates:  []ContainerHook{preFunc(prefix, "terminate", lifecycleID, 1), preFunc(prefix, "terminate", lifecycleID, 2)},
			PostTerminates: []ContainerHook{postFunc(prefix, "terminate", lifecycleID, 1), postFunc(prefix, "terminate", lifecycleID, 2)},
		}
	}

	defaultHooks := []LifecycleHooks{lifecycleHookFunc("default", 1), lifecycleHookFunc("default", 2)}
	userDefinedHooks := []LifecycleHooks{lifecycleHookFunc("user-defined", 1), lifecycleHookFunc("user-defined", 2), lifecycleHookFunc("user-defined", 3)}

	// call all the hooks in the right order, honouring the lifecycle

	def := Definition{
		lifecycleHooks: []LifecycleHooks{combineContainerHooks(defaultHooks, userDefinedHooks)},
	}
	err := def.creatingHook(context.Background())
	require.NoError(t, err)

	c := &Container{
		lifecycleHooks: def.lifecycleHooks,
	}

	err = c.createdHook(context.Background())
	require.NoError(t, err)
	err = c.startingHook(context.Background())
	require.NoError(t, err)
	err = c.startedHook(context.Background())
	require.NoError(t, err)
	err = c.readiedHook(context.Background())
	require.NoError(t, err)
	err = c.stoppingHook(context.Background())
	require.NoError(t, err)
	err = c.stoppedHook(context.Background())
	require.NoError(t, err)
	err = c.terminatingHook(context.Background())
	require.NoError(t, err)
	err = c.terminatedHook(context.Background())
	require.NoError(t, err)

	// assertions

	// There are 2 default container lifecycle hooks and 3 user-defined container lifecycle hooks.
	// Each lifecycle hook has 2 pre-create hooks and 2 post-create hooks.
	// That results in 16 hooks per lifecycle (8 defaults + 12 user-defined = 20)

	// There are 5 lifecycles (create, start, ready, stop, terminate),
	// but ready has only half of the hooks (it only has post), so we have 90 hooks in total.
	require.Len(t, prints, 90)

	// The order of the hooks is:
	// - pre-X hooks: first default (2*2), then user-defined (3*2)
	// - post-X hooks: first user-defined (3*2), then default (2*2)

	for i := range 5 {
		var hookType string
		// this is the particular order of execution for the hooks
		switch i {
		case 0:
			hookType = "create"
		case 1:
			hookType = "start"
		case 2:
			hookType = "ready"
		case 3:
			hookType = "stop"
		case 4:
			hookType = "terminate"
		}

		initialIndex := i * 20
		if i >= 2 {
			initialIndex -= 10
		}

		if hookType != "ready" {
			// default pre-hooks: 4 hooks
			require.Equal(t, fmt.Sprintf("[default] pre-%s hook 1.1", hookType), prints[initialIndex])
			require.Equal(t, fmt.Sprintf("[default] pre-%s hook 1.2", hookType), prints[initialIndex+1])
			require.Equal(t, fmt.Sprintf("[default] pre-%s hook 2.1", hookType), prints[initialIndex+2])
			require.Equal(t, fmt.Sprintf("[default] pre-%s hook 2.2", hookType), prints[initialIndex+3])

			// user-defined pre-hooks: 6 hooks
			require.Equal(t, fmt.Sprintf("[user-defined] pre-%s hook 1.1", hookType), prints[initialIndex+4])
			require.Equal(t, fmt.Sprintf("[user-defined] pre-%s hook 1.2", hookType), prints[initialIndex+5])
			require.Equal(t, fmt.Sprintf("[user-defined] pre-%s hook 2.1", hookType), prints[initialIndex+6])
			require.Equal(t, fmt.Sprintf("[user-defined] pre-%s hook 2.2", hookType), prints[initialIndex+7])
			require.Equal(t, fmt.Sprintf("[user-defined] pre-%s hook 3.1", hookType), prints[initialIndex+8])
			require.Equal(t, fmt.Sprintf("[user-defined] pre-%s hook 3.2", hookType), prints[initialIndex+9])
		}

		// user-defined post-hooks: 6 hooks
		require.Equal(t, fmt.Sprintf("[user-defined] post-%s hook 1.1", hookType), prints[initialIndex+10])
		require.Equal(t, fmt.Sprintf("[user-defined] post-%s hook 1.2", hookType), prints[initialIndex+11])
		require.Equal(t, fmt.Sprintf("[user-defined] post-%s hook 2.1", hookType), prints[initialIndex+12])
		require.Equal(t, fmt.Sprintf("[user-defined] post-%s hook 2.2", hookType), prints[initialIndex+13])
		require.Equal(t, fmt.Sprintf("[user-defined] post-%s hook 3.1", hookType), prints[initialIndex+14])
		require.Equal(t, fmt.Sprintf("[user-defined] post-%s hook 3.2", hookType), prints[initialIndex+15])

		// default post-hooks: 4 hooks
		require.Equal(t, fmt.Sprintf("[default] post-%s hook 1.1", hookType), prints[initialIndex+16])
		require.Equal(t, fmt.Sprintf("[default] post-%s hook 1.2", hookType), prints[initialIndex+17])
		require.Equal(t, fmt.Sprintf("[default] post-%s hook 2.1", hookType), prints[initialIndex+18])
		require.Equal(t, fmt.Sprintf("[default] post-%s hook 2.2", hookType), prints[initialIndex+19])
	}
}

func TestLifecycleHooks(t *testing.T) {
	prints := []string{}
	ctx := context.Background()

	opts := []ContainerCustomizer{
		WithImage(nginxAlpineImage),
		WithLifecycleHooks(LifecycleHooks{
			PreCreates: []DefinitionHook{
				func(_ context.Context, _ *Definition) error {
					prints = append(prints, "pre-create hook 1")
					return nil
				},
				func(_ context.Context, _ *Definition) error {
					prints = append(prints, "pre-create hook 2")
					return nil
				},
			},
			PostCreates: []ContainerHook{
				func(_ context.Context, _ *Container) error {
					prints = append(prints, "post-create hook 1")
					return nil
				},
				func(_ context.Context, _ *Container) error {
					prints = append(prints, "post-create hook 2")
					return nil
				},
			},
			PreStarts: []ContainerHook{
				func(_ context.Context, _ *Container) error {
					prints = append(prints, "pre-start hook 1")
					return nil
				},
				func(_ context.Context, _ *Container) error {
					prints = append(prints, "pre-start hook 2")
					return nil
				},
			},
			PostStarts: []ContainerHook{
				func(_ context.Context, _ *Container) error {
					prints = append(prints, "post-start hook 1")
					return nil
				},
				func(_ context.Context, _ *Container) error {
					prints = append(prints, "post-start hook 2")
					return nil
				},
			},
			PostReadies: []ContainerHook{
				func(_ context.Context, _ *Container) error {
					prints = append(prints, "post-ready hook 1")
					return nil
				},
				func(_ context.Context, _ *Container) error {
					prints = append(prints, "post-ready hook 2")
					return nil
				},
			},
			PreStops: []ContainerHook{
				func(_ context.Context, _ *Container) error {
					prints = append(prints, "pre-stop hook 1")
					return nil
				},
				func(_ context.Context, _ *Container) error {
					prints = append(prints, "pre-stop hook 2")
					return nil
				},
			},
			PostStops: []ContainerHook{
				func(_ context.Context, _ *Container) error {
					prints = append(prints, "post-stop hook 1")
					return nil
				},
				func(_ context.Context, _ *Container) error {
					prints = append(prints, "post-stop hook 2")
					return nil
				},
			},
			PreTerminates: []ContainerHook{
				func(_ context.Context, _ *Container) error {
					prints = append(prints, "pre-terminate hook 1")
					return nil
				},
				func(_ context.Context, _ *Container) error {
					prints = append(prints, "pre-terminate hook 2")
					return nil
				},
			},
			PostTerminates: []ContainerHook{
				func(_ context.Context, _ *Container) error {
					prints = append(prints, "post-terminate hook 1")
					return nil
				},
				func(_ context.Context, _ *Container) error {
					prints = append(prints, "post-terminate hook 2")
					return nil
				},
			},
		}),
	}

	c, err := Run(ctx, opts...)
	CleanupContainer(t, c)
	require.NoError(t, err)
	require.NotNil(t, c)

	err = c.Stop(ctx, StopTimeout(1*time.Second))
	require.NoError(t, err)

	err = c.Start(ctx)
	require.NoError(t, err)

	err = c.Terminate(ctx)
	require.NoError(t, err)

	lifecycleHooksIsHonouredFn(t, prints)
}

func lifecycleHooksIsHonouredFn(t *testing.T, prints []string) {
	t.Helper()

	expects := []string{
		"pre-create hook 1",
		"pre-create hook 2",
		"post-create hook 1",
		"post-create hook 2",
		"pre-start hook 1",
		"pre-start hook 2",
		"post-start hook 1",
		"post-start hook 2",
		"post-ready hook 1",
		"post-ready hook 2",
		"pre-stop hook 1",
		"pre-stop hook 2",
		"post-stop hook 1",
		"post-stop hook 2",
		"pre-start hook 1",
		"pre-start hook 2",
		"post-start hook 1",
		"post-start hook 2",
		"post-ready hook 1",
		"post-ready hook 2",
		// Terminate currently calls stop to ensure that child containers are stopped.
		"pre-stop hook 1",
		"pre-stop hook 2",
		"post-stop hook 1",
		"post-stop hook 2",
		"pre-terminate hook 1",
		"pre-terminate hook 2",
		"post-terminate hook 1",
		"post-terminate hook 2",
	}

	require.Equal(t, expects, prints)
}
