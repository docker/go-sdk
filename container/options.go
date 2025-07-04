package container

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/docker/docker/api/types/container"
	apinetwork "github.com/docker/docker/api/types/network"
	"github.com/docker/go-sdk/client"
	"github.com/docker/go-sdk/container/exec"
	"github.com/docker/go-sdk/container/wait"
	"github.com/docker/go-sdk/network"
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
func WithDockerClient(dockerClient *client.Client) CustomizeDefinitionOption {
	return func(def *Definition) error {
		def.dockerClient = dockerClient

		return nil
	}
}

// WithConfigModifier allows to override the default container config
func WithConfigModifier(modifier func(config *container.Config)) CustomizeDefinitionOption {
	return func(def *Definition) error {
		def.configModifier = modifier

		return nil
	}
}

// WithEndpointSettingsModifier allows to override the default endpoint settings
func WithEndpointSettingsModifier(modifier func(settings map[string]*apinetwork.EndpointSettings)) CustomizeDefinitionOption {
	return func(def *Definition) error {
		def.endpointSettingsModifier = modifier

		return nil
	}
}

// WithEnv sets the environment variables for a container.
// If the environment variable already exists, it will be overridden.
func WithEnv(envs map[string]string) CustomizeDefinitionOption {
	return func(def *Definition) error {
		if def.env == nil {
			def.env = map[string]string{}
		}

		maps.Copy(def.env, envs)
		return nil
	}
}

// WithHostConfigModifier allows to override the default host config
func WithHostConfigModifier(modifier func(hostConfig *container.HostConfig)) CustomizeDefinitionOption {
	return func(def *Definition) error {
		def.hostConfigModifier = modifier

		return nil
	}
}

// WithName will set the name of the container.
func WithName(containerName string) CustomizeDefinitionOption {
	return func(def *Definition) error {
		if containerName == "" {
			return ErrReuseEmptyName
		}
		def.name = containerName
		return nil
	}
}

// WithNoStart will prevent the container from being started after creation.
func WithNoStart() CustomizeDefinitionOption {
	return func(def *Definition) error {
		def.started = false
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
		def.imageSubstitutors = fn

		return nil
	}
}

// WithNetwork reuses an already existing network, attaching the container to it.
// Finally it sets the network alias on that network to the given alias.
func WithNetwork(aliases []string, nw *network.Network) CustomizeDefinitionOption {
	return WithNetworkName(aliases, nw.Name())
}

// WithNetworkName attachs a container to an already existing network, by its name.
// If the network is not "bridge", it sets the network alias on that network
// to the given alias, else, it returns an error. This is because network-scoped alias
// is supported only for containers in user defined networks.
func WithNetworkName(aliases []string, networkName string) CustomizeDefinitionOption {
	return func(def *Definition) error {
		if networkName == "bridge" {
			return errors.New("network-scoped aliases are supported only for containers in user defined networks")
		}

		// attaching to the network because it was created with success or it already existed.
		def.networks = append(def.networks, networkName)

		if def.networkAliases == nil {
			def.networkAliases = make(map[string][]string)
		}
		def.networkAliases[networkName] = aliases

		return nil
	}
}

// WithBridgeNetwork attachs a container to the "bridge" network.
// There is no need to set the network alias, as it is not supported for the "bridge" network.
func WithBridgeNetwork() CustomizeDefinitionOption {
	return func(def *Definition) error {
		def.networks = append(def.networks, "bridge")
		return nil
	}
}

// WithNewNetwork creates a new network with random name and customizers, and attaches the container to it.
// Finally it sets the network alias on that network to the given alias.
func WithNewNetwork(ctx context.Context, aliases []string, opts ...network.Option) CustomizeDefinitionOption {
	return func(def *Definition) error {
		newNetwork, err := network.New(ctx, opts...)
		if err != nil {
			return fmt.Errorf("new network: %w", err)
		}

		networkName := newNetwork.Name()

		// attaching to the network because it was created with success or it already existed.
		def.networks = append(def.networks, networkName)

		if def.networkAliases == nil {
			def.networkAliases = make(map[string][]string)
		}
		def.networkAliases[networkName] = aliases

		return nil
	}
}

// Executable represents an executable command to be sent to a container, including options,
// as part of the different lifecycle hooks.
type Executable interface {
	AsCommand() []string
	// Options can container two different types of options:
	// - Docker's ExecConfigs (WithUser, WithWorkingDir, WithEnv, etc.)
	// - SDK's ProcessOptions (i.e. Multiplexed response)
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

		def.lifecycleHooks = append(def.lifecycleHooks, startupCommandsHook)

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

		def.lifecycleHooks = append(def.lifecycleHooks, LifecycleHooks{
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
		def.waitingFor = wait.ForAll(strategies...).WithDeadline(deadline)

		return nil
	}
}

// WithAdditionalWaitStrategyAndDeadline appends the wait strategy for a container, including deadline
func WithAdditionalWaitStrategyAndDeadline(deadline time.Duration, strategies ...wait.Strategy) CustomizeDefinitionOption {
	return func(def *Definition) error {
		if def.waitingFor == nil {
			def.waitingFor = wait.ForAll(strategies...).WithDeadline(deadline)
			return nil
		}

		wss := make([]wait.Strategy, 0, len(strategies)+1)
		wss = append(wss, def.waitingFor)
		wss = append(wss, strategies...)

		def.waitingFor = wait.ForAll(wss...).WithDeadline(deadline)

		return nil
	}
}

// WithAlwaysPull will pull the image before starting the container
func WithAlwaysPull() CustomizeDefinitionOption {
	return func(def *Definition) error {
		def.alwaysPullImage = true
		return nil
	}
}

// WithImagePlatform sets the platform for a container
func WithImagePlatform(platform string) CustomizeDefinitionOption {
	return func(def *Definition) error {
		def.imagePlatform = platform
		return nil
	}
}

// WithEntrypoint completely replaces the entrypoint of a container
func WithEntrypoint(entrypoint ...string) CustomizeDefinitionOption {
	return func(def *Definition) error {
		def.entrypoint = entrypoint
		return nil
	}
}

// WithEntrypointArgs appends the entrypoint arguments to the entrypoint of a container
func WithEntrypointArgs(entrypointArgs ...string) CustomizeDefinitionOption {
	return func(def *Definition) error {
		def.entrypoint = append(def.entrypoint, entrypointArgs...)
		return nil
	}
}

// WithExposedPorts appends the ports to the exposed ports for a container
func WithExposedPorts(ports ...string) CustomizeDefinitionOption {
	return func(def *Definition) error {
		def.exposedPorts = append(def.exposedPorts, ports...)
		return nil
	}
}

// WithCmd completely replaces the command for a container
func WithCmd(cmd ...string) CustomizeDefinitionOption {
	return func(def *Definition) error {
		def.cmd = cmd
		return nil
	}
}

// WithCmdArgs appends the command arguments to the command for a container
func WithCmdArgs(cmdArgs ...string) CustomizeDefinitionOption {
	return func(def *Definition) error {
		def.cmd = append(def.cmd, cmdArgs...)
		return nil
	}
}

// WithLabels appends the labels to the labels for a container
func WithLabels(labels map[string]string) CustomizeDefinitionOption {
	return func(def *Definition) error {
		if def.labels == nil {
			def.labels = make(map[string]string)
		}

		maps.Copy(def.labels, labels)
		return nil
	}
}

// WithLifecycleHooks completely replaces the lifecycle hooks for a container
func WithLifecycleHooks(hooks ...LifecycleHooks) CustomizeDefinitionOption {
	return func(def *Definition) error {
		def.lifecycleHooks = hooks
		return nil
	}
}

// WithAdditionalLifecycleHooks appends lifecycle hooks to the existing ones for a container
func WithAdditionalLifecycleHooks(hooks ...LifecycleHooks) CustomizeDefinitionOption {
	return func(def *Definition) error {
		def.lifecycleHooks = append(def.lifecycleHooks, hooks...)
		return nil
	}
}

// WithFiles appends the files to the files for a container
func WithFiles(files ...File) CustomizeDefinitionOption {
	return func(def *Definition) error {
		def.files = append(def.files, files...)
		return nil
	}
}
