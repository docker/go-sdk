package dockercontainer

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/docker/go-sdk/dockerclient"
)

// defaultPreCreateHook is a hook that will apply the default configuration to the container
var defaultPreCreateHook = func(dockerClient *dockerclient.Client, dockerInput *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig) LifecycleHooks {
	return LifecycleHooks{
		PreCreates: []DefinitionHook{
			func(ctx context.Context, def *Definition) error {
				return preCreateContainerHook(ctx, dockerClient, def, dockerInput, hostConfig, networkingConfig)
			},
		},
	}
}

// defaultCopyFileToContainerHook is a hook that will copy files to the container after it's created
// but before it's started
var defaultCopyFileToContainerHook = func(files []File) LifecycleHooks {
	return LifecycleHooks{
		PostCreates: []ContainerHook{
			// copy files to container after it's created
			func(ctx context.Context, c *Container) error {
				for _, f := range files {
					if err := f.validate(); err != nil {
						return fmt.Errorf("invalid file: %w", err)
					}

					var bs []byte
					var err error
					if f.Reader != nil {
						// Bytes takes precedence over HostFilePath
						bs, err = io.ReadAll(f.Reader)
						if err != nil {
							return fmt.Errorf("read all: %w", err)
						}
					} else {
						// no reader, read from host path
						bs, err = os.ReadFile(f.HostPath)
						if err != nil {
							return fmt.Errorf("read file: %w", err)
						}
					}

					err = c.CopyToContainer(ctx, bs, f.ContainerPath, f.Mode)
					if err != nil {
						return fmt.Errorf("copy to container at %s: %w", f.ContainerPath, err)
					}
				}

				return nil
			},
		},
	}
}

// defaultLogConsumersHook is a hook that will start log consumers after the container is started
var defaultLogConsumersHook = func(cfg *LogConsumerConfig) LifecycleHooks {
	return LifecycleHooks{
		PostStarts: []ContainerHook{
			// Produce logs sending details to the log consumers.
			// See combineContainerHooks for the order of execution.
			func(ctx context.Context, c *Container) error {
				if cfg == nil || len(cfg.Consumers) == 0 {
					return nil
				}

				c.resetConsumers(cfg.Consumers)

				return c.startLogProduction(ctx, cfg.Opts...)
			},
		},
		PostStops: []ContainerHook{
			// Stop the log production.
			// See combineContainerHooks for the order of execution.
			func(_ context.Context, c *Container) error {
				if cfg == nil || len(cfg.Consumers) == 0 {
					return nil
				}

				return c.stopLogProduction()
			},
		},
	}
}

// defaultReadinessHook is a hook that will wait for the container to be ready
var defaultReadinessHook = func() LifecycleHooks {
	return LifecycleHooks{
		PostStarts: []ContainerHook{
			// wait for the container to be ready
			func(ctx context.Context, c *Container) error {
				// if a Wait Strategy has been specified, wait before returning
				if c.waitingFor != nil {
					c.logger.Info("Waiting for container to be ready", "containerID", c.ShortID(), "image", c.Image())
					if err := c.waitingFor.WaitUntilReady(ctx, c); err != nil {
						return fmt.Errorf("wait until ready: %w", err)
					}
				}

				c.isRunning = true

				return nil
			},
		},
	}
}

// creatingHook is a hook that will be called before a container is created.
func (def *Definition) creatingHook(ctx context.Context) error {
	return def.applyLifecycleHooks(func(lifecycleHooks LifecycleHooks) error {
		return applyDefinitionHooks(ctx, lifecycleHooks.PreCreates, def)
	})
}

// createdHook is a hook that will be called after a container is created.
func (c *Container) createdHook(ctx context.Context) error {
	return c.applyLifecycleHooks(ctx, false, func(lifecycleHooks LifecycleHooks) error {
		return applyContainerHooks(ctx, lifecycleHooks.PostCreates, c)
	})
}

func mergePortBindings(configPortMap, exposedPortMap nat.PortMap, exposedPorts []string) nat.PortMap {
	if exposedPortMap == nil {
		exposedPortMap = make(map[nat.Port][]nat.PortBinding)
	}

	mappedPorts := make(map[string]struct{}, len(exposedPorts))
	for _, p := range exposedPorts {
		p = strings.Split(p, "/")[0]
		mappedPorts[p] = struct{}{}
	}

	for k, v := range configPortMap {
		if _, ok := mappedPorts[k.Port()]; ok {
			exposedPortMap[k] = v
		}
	}
	return exposedPortMap
}

func preCreateContainerHook(ctx context.Context, dockerClient *dockerclient.Client, def *Definition, dockerInput *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig) error {
	endpointSettings := map[string]*network.EndpointSettings{}

	// Docker allows only one network to be specified during container creation
	// If there is more than one network specified in the request container should be attached to them
	// once it is created. We will take a first network if any specified in the request and use it to create container
	if len(def.networks) > 0 {
		attachContainerTo := def.networks[0]

		nwInspect, err := dockerClient.NetworkInspect(ctx, def.networks[0], network.InspectOptions{
			Verbose: true,
		})
		if err != nil {
			return fmt.Errorf("network inspect: %w", err)
		}

		aliases := []string{}
		if _, ok := def.networkAliases[attachContainerTo]; ok {
			aliases = def.networkAliases[attachContainerTo]
		}
		endpointSetting := network.EndpointSettings{
			Aliases:   aliases,
			NetworkID: nwInspect.ID,
		}
		endpointSettings[attachContainerTo] = &endpointSetting

	}

	if def.configModifier != nil {
		def.configModifier(dockerInput)
	}

	if def.hostConfigModifier != nil {
		def.hostConfigModifier(hostConfig)
	}

	if def.endpointSettingsModifier != nil {
		def.endpointSettingsModifier(endpointSettings)
	}

	networkingConfig.EndpointsConfig = endpointSettings

	exposedPorts := def.exposedPorts
	// this check must be done after the pre-creation Modifiers are called, so the network mode is already set
	if len(exposedPorts) == 0 && !hostConfig.NetworkMode.IsContainer() {
		hostConfig.PublishAllPorts = true
	}

	exposedPortSet, exposedPortMap, err := nat.ParsePortSpecs(exposedPorts)
	if err != nil {
		return err
	}

	dockerInput.ExposedPorts = exposedPortSet

	// only exposing those ports automatically if the container request exposes zero ports and the container does not run in a container network
	if len(exposedPorts) == 0 && !hostConfig.NetworkMode.IsContainer() {
		hostConfig.PortBindings = exposedPortMap
	} else {
		hostConfig.PortBindings = mergePortBindings(hostConfig.PortBindings, exposedPortMap, def.exposedPorts)
	}

	return nil
}
