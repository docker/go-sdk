package dockercontainer

import (
	"context"
	"errors"
	"fmt"

	"github.com/containerd/errdefs"
	"github.com/containerd/platforms"
	specs "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-sdk/dockerclient"
	"github.com/docker/go-sdk/dockerimage"
)

// Create fulfils a request for a container without starting it
func Create(ctx context.Context, opts ...ContainerCustomizer) (*Container, error) {
	def := Definition{
		Env: make(map[string]string),
	}

	for _, opt := range opts {
		if err := opt.Customize(&def); err != nil {
			return nil, fmt.Errorf("customize: %w", err)
		}
	}

	if def.image == "" {
		return nil, errors.New("image is required")
	}

	if def.DockerClient == nil {
		// use the default docker client
		cli, err := dockerclient.New(ctx)
		if err != nil {
			return nil, fmt.Errorf("new docker client: %w", err)
		}
		def.DockerClient = cli
	}

	env := []string{}
	for envKey, envVar := range def.Env {
		env = append(env, envKey+"="+envVar)
	}

	if def.Labels == nil {
		def.Labels = make(map[string]string)
	}

	defaultHooks := []LifecycleHooks{
		DefaultLoggingHook(def.DockerClient.Logger()),
	}

	for _, is := range def.ImageSubstitutors {
		modifiedTag, err := is.Substitute(def.image)
		if err != nil {
			return nil, fmt.Errorf("failed to substitute image %s with %s: %w", def.image, is.Description(), err)
		}

		if modifiedTag != def.image {
			def.DockerClient.Logger().Info("Replacing image", "description", is.Description(), "from", def.image, "to", modifiedTag)
			def.image = modifiedTag
		}
	}

	var platform *specs.Platform

	if def.ImagePlatform != "" {
		p, err := platforms.Parse(def.ImagePlatform)
		if err != nil {
			return nil, fmt.Errorf("invalid platform %s: %w", def.ImagePlatform, err)
		}
		platform = &p
	}

	var shouldPullImage bool

	if def.AlwaysPullImage {
		shouldPullImage = true // If requested always attempt to pull image
	} else {
		img, err := def.DockerClient.Client().ImageInspect(ctx, def.image)
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
			Platform: def.ImagePlatform, // may be empty
		}
		if err := dockerimage.Pull(ctx, def.DockerClient, def.image, pullOpt); err != nil {
			return nil, err
		}
	}

	// Add the labels that identify this as a container created by the SDK.
	AddSDKLabels(def.Labels)

	dockerInput := &container.Config{
		Entrypoint: def.Entrypoint,
		Image:      def.image,
		Env:        env,
		Labels:     def.Labels,
		Cmd:        def.Cmd,
	}

	hostConfig := &container.HostConfig{}

	networkingConfig := &network.NetworkingConfig{}

	// default hooks include logger hook and pre-create hook
	defaultHooks = append(defaultHooks,
		defaultPreCreateHook(def.DockerClient, dockerInput, hostConfig, networkingConfig),
		defaultCopyFileToContainerHook(def.Files),
		defaultLogConsumersHook(def.LogConsumerCfg),
		defaultReadinessHook(),
	)

	// Combine with the original LifecycleHooks to avoid duplicate logging hooks.
	origLifecycleHooks := def.LifecycleHooks
	def.LifecycleHooks = []LifecycleHooks{
		combineContainerHooks(defaultHooks, origLifecycleHooks),
	}

	err := def.creatingHook(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := def.DockerClient.Client().ContainerCreate(ctx, dockerInput, hostConfig, networkingConfig, platform, def.Name)
	if err != nil {
		return nil, fmt.Errorf("container create: %w", err)
	}

	// If there is more than one network specified in the request attach newly created container to them one by one
	if len(def.Networks) > 1 {
		for _, n := range def.Networks[1:] {
			nwInspect, err := def.DockerClient.Client().NetworkInspect(ctx, n, network.InspectOptions{
				Verbose: true,
			})
			if err != nil {
				return nil, fmt.Errorf("network inspect: %w", err)
			}

			endpointSetting := network.EndpointSettings{
				Aliases: def.NetworkAliases[n],
			}
			err = def.DockerClient.Client().NetworkConnect(ctx, nwInspect.ID, resp.ID, &endpointSetting)
			if err != nil {
				return nil, fmt.Errorf("network connect: %w", err)
			}
		}
	}

	// This should match the fields set in ContainerFromDockerResponse.
	ctr := &Container{
		dockerClient:   def.DockerClient,
		ID:             resp.ID,
		shortID:        resp.ID[:12],
		WaitingFor:     def.WaitingFor,
		Image:          def.image,
		exposedPorts:   def.ExposedPorts,
		logger:         def.DockerClient.Logger(),
		lifecycleHooks: def.LifecycleHooks,
	}

	if err = ctr.createdHook(ctx); err != nil {
		// Return the container to allow caller to clean up.
		return ctr, fmt.Errorf("created hook: %w", err)
	}

	return ctr, nil
}
