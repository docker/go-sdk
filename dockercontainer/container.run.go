package dockercontainer

import (
	"context"
	"fmt"

	"github.com/containerd/errdefs"
	"github.com/containerd/platforms"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-sdk/dockerclient"
	"github.com/docker/go-sdk/dockerimage"
)

// Run is a convenience function that creates a new container and starts it.
// By default, the container is started after creation, unless requested otherwise
// using the [WithNoStart] option.
func Run(ctx context.Context, opts ...ContainerCustomizer) (*Container, error) {
	def := Definition{
		env:     make(map[string]string),
		started: true,
	}

	for _, opt := range opts {
		if err := opt.Customize(&def); err != nil {
			return nil, fmt.Errorf("customize: %w", err)
		}
	}

	if err := def.validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	if def.dockerClient == nil {
		// use the default docker client
		cli, err := dockerclient.New(ctx)
		if err != nil {
			return nil, fmt.Errorf("new docker client: %w", err)
		}
		def.dockerClient = cli
	}

	env := []string{}
	for envKey, envVar := range def.env {
		env = append(env, envKey+"="+envVar)
	}

	if def.labels == nil {
		def.labels = make(map[string]string)
	}

	defaultHooks := []LifecycleHooks{
		DefaultLoggingHook(def.dockerClient.Logger()),
	}

	for _, is := range def.imageSubstitutors {
		modifiedTag, err := is.Substitute(def.image)
		if err != nil {
			return nil, fmt.Errorf("failed to substitute image %s with %s: %w", def.image, is.Description(), err)
		}

		if modifiedTag != def.image {
			def.dockerClient.Logger().Info("Replacing image", "description", is.Description(), "from", def.image, "to", modifiedTag)
			def.image = modifiedTag
		}
	}

	var platform *platforms.Platform

	if def.imagePlatform != "" {
		p, err := platforms.Parse(def.imagePlatform)
		if err != nil {
			return nil, fmt.Errorf("invalid platform %s: %w", def.imagePlatform, err)
		}
		platform = &p
	}

	var shouldPullImage bool

	if def.alwaysPullImage {
		shouldPullImage = true // If requested always attempt to pull image
	} else {
		img, err := def.dockerClient.ImageInspect(ctx, def.image)
		if err != nil {
			if !errdefs.IsNotFound(err) {
				return nil, err
			}
			shouldPullImage = true
		}
		if platform != nil && (img.Architecture != platform.Architecture || img.Os != platform.OS) {
			shouldPullImage = true
		}
	}

	if shouldPullImage {
		pullOpt := image.PullOptions{
			Platform: def.imagePlatform, // may be empty
		}
		if err := dockerimage.Pull(ctx, def.dockerClient, def.image, pullOpt); err != nil {
			return nil, err
		}
	}

	// Add the labels that identify this as a container created by the SDK.
	dockerclient.AddSDKLabels(def.labels)

	dockerInput := &container.Config{
		Entrypoint: def.entrypoint,
		Image:      def.image,
		Env:        env,
		Labels:     def.labels,
		Cmd:        def.cmd,
	}

	hostConfig := &container.HostConfig{}

	networkingConfig := &network.NetworkingConfig{}

	// default hooks include logger hook and pre-create hook
	defaultHooks = append(defaultHooks,
		defaultPreCreateHook(def.dockerClient, dockerInput, hostConfig, networkingConfig),
		defaultCopyFileToContainerHook(def.files),
		defaultLogConsumersHook(def.logConsumerCfg),
		defaultReadinessHook(),
	)

	// Combine with the original LifecycleHooks to avoid duplicate logging hooks.
	origLifecycleHooks := def.lifecycleHooks
	def.lifecycleHooks = []LifecycleHooks{
		combineContainerHooks(defaultHooks, origLifecycleHooks),
	}

	err := def.creatingHook(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := def.dockerClient.ContainerCreate(ctx, dockerInput, hostConfig, networkingConfig, platform, def.name)
	if err != nil {
		return nil, fmt.Errorf("container create: %w", err)
	}

	// If there is more than one network specified in the request attach newly created container to them one by one
	if len(def.networks) > 1 {
		for _, n := range def.networks[1:] {
			nwInspect, err := def.dockerClient.NetworkInspect(ctx, n, network.InspectOptions{
				Verbose: true,
			})
			if err != nil {
				return nil, fmt.Errorf("network inspect: %w", err)
			}

			endpointSetting := network.EndpointSettings{
				Aliases: def.networkAliases[n],
			}
			err = def.dockerClient.NetworkConnect(ctx, nwInspect.ID, resp.ID, &endpointSetting)
			if err != nil {
				return nil, fmt.Errorf("network connect: %w", err)
			}
		}
	}

	// This should match the fields set in ContainerFromDockerResponse.
	ctr := &Container{
		dockerClient:   def.dockerClient,
		containerID:    resp.ID,
		shortID:        resp.ID[:12],
		waitingFor:     def.waitingFor,
		image:          def.image,
		exposedPorts:   def.exposedPorts,
		logger:         def.dockerClient.Logger(),
		lifecycleHooks: def.lifecycleHooks,
	}

	if err = ctr.createdHook(ctx); err != nil {
		// Return the container to allow caller to clean up.
		return ctr, fmt.Errorf("created hook: %w", err)
	}

	if def.started {
		if err := ctr.Start(ctx); err != nil {
			return ctr, fmt.Errorf("start container: %w", err)
		}
	}

	return ctr, nil
}
