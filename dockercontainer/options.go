package dockercontainer

import (
	"context"
	"errors"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-sdk/dockerclient"
	"github.com/docker/go-sdk/dockercontainer/exec"
	"github.com/docker/go-sdk/dockercontainer/wait"
)

var ErrReuseEmptyName = errors.New("with reuse option a container name mustn't be empty")

// ContainerCustomizer is an interface that can be used to configure the container
// definition. The passed definition is merged with the default one.
type ContainerCustomizer interface {
	Customize(def *Definition) error
}

// CustomizeDefinitionOption is a type that can be used to configure the container definition.
// The passed definition is merged with the default one.
type CustomizeDefinitionOption func(def *Definition) error

// Customize implements the ContainerCustomizer interface.
func (opt CustomizeDefinitionOption) Customize(def *Definition) error {
	return opt(def)
}

// WithDockerClient sets the docker client for a container
func WithDockerClient(dockerClient *dockerclient.Client) CustomizeDefinitionOption {
	return func(def *Definition) error {
		def.DockerClient = dockerClient

		return nil
	}
}

// WithConfigModifier allows to override the default container config
func WithConfigModifier(modifier func(config *container.Config)) CustomizeDefinitionOption {
	return func(def *Definition) error {
		def.ConfigModifier = modifier

		return nil
	}
}

// WithEndpointSettingsModifier allows to override the default endpoint settings
func WithEndpointSettingsModifier(modifier func(settings map[string]*network.EndpointSettings)) CustomizeDefinitionOption {
	return func(def *Definition) error {
		def.EndpointSettingsModifier = modifier

		return nil
	}
}

// WithEnv sets the environment variables for a container.
// If the environment variable already exists, it will be overridden.
func WithEnv(envs map[string]string) CustomizeDefinitionOption {
	return func(def *Definition) error {
		if def.Env == nil {
			def.Env = map[string]string{}
		}

		for key, val := range envs {
			def.Env[key] = val
		}

		return nil
	}
}

// WithHostConfigModifier allows to override the default host config
func WithHostConfigModifier(modifier func(hostConfig *container.HostConfig)) CustomizeDefinitionOption {
	return func(def *Definition) error {
		def.HostConfigModifier = modifier

		return nil
	}
}

// WithName will set the name of the container.
func WithName(containerName string) CustomizeDefinitionOption {
	return func(def *Definition) error {
		if containerName == "" {
			return ErrReuseEmptyName
		}
		def.Name = containerName
		return nil
	}
}

// WithNoStart will prevent the container from being started after creation.
func WithNoStart() CustomizeDefinitionOption {
	return func(def *Definition) error {
		def.Started = false
		return nil
	}
}

// WithImage sets the image for a container
func WithImage(image string) CustomizeDefinitionOption {
	return func(def *Definition) error {
		def.image = image

		return nil
	}
}

// WithImageSubstitutors sets the image substitutors for a container
func WithImageSubstitutors(fn ...ImageSubstitutor) CustomizeDefinitionOption {
	return func(def *Definition) error {
		def.ImageSubstitutors = fn

		return nil
	}
}

// WithLogConsumers sets the log consumers for a container
func WithLogConsumers(consumer ...LogConsumer) CustomizeDefinitionOption {
	return func(def *Definition) error {
		if def.LogConsumerCfg == nil {
			def.LogConsumerCfg = &LogConsumerConfig{}
		}

		def.LogConsumerCfg.Consumers = consumer
		return nil
	}
}

// WithLogConsumerConfig sets the log consumer config for a container.
// Beware that this option completely replaces the existing log consumer config,
// including the log consumers and the log production options,
// so it should be used with care.
func WithLogConsumerConfig(config *LogConsumerConfig) CustomizeDefinitionOption {
	return func(def *Definition) error {
		def.LogConsumerCfg = config
		return nil
	}
}

// Executable represents an executable command to be sent to a container, including options,
// as part of the different lifecycle hooks.
type Executable interface {
	AsCommand() []string
	// Options can container two different types of options:
	// - Docker's ExecConfigs (WithUser, WithWorkingDir, WithEnv, etc.)
	// - testcontainers' ProcessOptions (i.e. Multiplexed response)
	Options() []exec.ProcessOption
}

// WithStartupCommand will execute the command representation of each Executable into the container.
// It will leverage the container lifecycle hooks to call the command right after the container
// is started.
func WithStartupCommand(execs ...Executable) CustomizeDefinitionOption {
	return func(def *Definition) error {
		startupCommandsHook := LifecycleHooks{
			PostStarts: []ContainerHook{},
		}

		for _, exec := range execs {
			execFn := func(ctx context.Context, c *Container) error {
				_, _, err := c.Exec(ctx, exec.AsCommand(), exec.Options()...)
				return err
			}

			startupCommandsHook.PostStarts = append(startupCommandsHook.PostStarts, execFn)
		}

		def.LifecycleHooks = append(def.LifecycleHooks, startupCommandsHook)

		return nil
	}
}

// WithAfterReadyCommand will execute the command representation of each Executable into the container.
// It will leverage the container lifecycle hooks to call the command right after the container
// is ready.
func WithAfterReadyCommand(execs ...Executable) CustomizeDefinitionOption {
	return func(def *Definition) error {
		postReadiesHook := []ContainerHook{}

		for _, exec := range execs {
			execFn := func(ctx context.Context, c *Container) error {
				_, _, err := c.Exec(ctx, exec.AsCommand(), exec.Options()...)
				return err
			}

			postReadiesHook = append(postReadiesHook, execFn)
		}

		def.LifecycleHooks = append(def.LifecycleHooks, LifecycleHooks{
			PostReadies: postReadiesHook,
		})

		return nil
	}
}

// WithWaitStrategy replaces the wait strategy for a container, using 60 seconds as deadline
func WithWaitStrategy(strategies ...wait.Strategy) CustomizeDefinitionOption {
	return WithWaitStrategyAndDeadline(60*time.Second, strategies...)
}

// WithAdditionalWaitStrategy appends the wait strategy for a container, using 60 seconds as deadline
func WithAdditionalWaitStrategy(strategies ...wait.Strategy) CustomizeDefinitionOption {
	return WithAdditionalWaitStrategyAndDeadline(60*time.Second, strategies...)
}

// WithWaitStrategyAndDeadline replaces the wait strategy for a container, including deadline
func WithWaitStrategyAndDeadline(deadline time.Duration, strategies ...wait.Strategy) CustomizeDefinitionOption {
	return func(def *Definition) error {
		def.WaitingFor = wait.ForAll(strategies...).WithDeadline(deadline)

		return nil
	}
}

// WithAdditionalWaitStrategyAndDeadline appends the wait strategy for a container, including deadline
func WithAdditionalWaitStrategyAndDeadline(deadline time.Duration, strategies ...wait.Strategy) CustomizeDefinitionOption {
	return func(def *Definition) error {
		if def.WaitingFor == nil {
			def.WaitingFor = wait.ForAll(strategies...).WithDeadline(deadline)
			return nil
		}

		wss := make([]wait.Strategy, 0, len(strategies)+1)
		wss = append(wss, def.WaitingFor)
		wss = append(wss, strategies...)

		def.WaitingFor = wait.ForAll(wss...).WithDeadline(deadline)

		return nil
	}
}

// WithAlwaysPull will pull the image before starting the container
func WithAlwaysPull() CustomizeDefinitionOption {
	return func(def *Definition) error {
		def.AlwaysPullImage = true
		return nil
	}
}

// WithImagePlatform sets the platform for a container
func WithImagePlatform(platform string) CustomizeDefinitionOption {
	return func(def *Definition) error {
		def.ImagePlatform = platform
		return nil
	}
}

// WithEntrypoint completely replaces the entrypoint of a container
func WithEntrypoint(entrypoint ...string) CustomizeDefinitionOption {
	return func(def *Definition) error {
		def.Entrypoint = entrypoint
		return nil
	}
}

// WithEntrypointArgs appends the entrypoint arguments to the entrypoint of a container
func WithEntrypointArgs(entrypointArgs ...string) CustomizeDefinitionOption {
	return func(def *Definition) error {
		def.Entrypoint = append(def.Entrypoint, entrypointArgs...)
		return nil
	}
}

// WithExposedPorts appends the ports to the exposed ports for a container
func WithExposedPorts(ports ...string) CustomizeDefinitionOption {
	return func(def *Definition) error {
		def.ExposedPorts = append(def.ExposedPorts, ports...)
		return nil
	}
}

// WithCmd completely replaces the command for a container
func WithCmd(cmd ...string) CustomizeDefinitionOption {
	return func(def *Definition) error {
		def.Cmd = cmd
		return nil
	}
}

// WithCmdArgs appends the command arguments to the command for a container
func WithCmdArgs(cmdArgs ...string) CustomizeDefinitionOption {
	return func(def *Definition) error {
		def.Cmd = append(def.Cmd, cmdArgs...)
		return nil
	}
}

// WithLabels appends the labels to the labels for a container
func WithLabels(labels map[string]string) CustomizeDefinitionOption {
	return func(def *Definition) error {
		if def.Labels == nil {
			def.Labels = make(map[string]string)
		}
		for k, v := range labels {
			def.Labels[k] = v
		}
		return nil
	}
}

// WithLifecycleHooks completely replaces the lifecycle hooks for a container
func WithLifecycleHooks(hooks ...LifecycleHooks) CustomizeDefinitionOption {
	return func(def *Definition) error {
		def.LifecycleHooks = hooks
		return nil
	}
}

// WithAdditionalLifecycleHooks appends lifecycle hooks to the existing ones for a container
func WithAdditionalLifecycleHooks(hooks ...LifecycleHooks) CustomizeDefinitionOption {
	return func(def *Definition) error {
		def.LifecycleHooks = append(def.LifecycleHooks, hooks...)
		return nil
	}
}

// WithFiles appends the files to the files for a container
func WithFiles(files ...File) CustomizeDefinitionOption {
	return func(def *Definition) error {
		def.Files = append(def.Files, files...)
		return nil
	}
}
