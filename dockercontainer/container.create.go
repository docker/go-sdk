package dockercontainer

import (
	"context"
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

// CreateContainer fulfils a request for a container without starting it
func CreateContainer(ctx context.Context, def Definition) (ctr *Container, err error) {
	imageName := def.Image

	env := []string{}
	for envKey, envVar := range def.Env {
		env = append(env, envKey+"="+envVar)
	}

	if def.Labels == nil {
		def.Labels = make(map[string]string)
	}

	var platform *specs.Platform

	dockerClient, err := dockerclient.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("new docker client: %w", err)
	}

	defaultHooks := []LifecycleHooks{
		DefaultLoggingHook(dockerClient.Logger()),
	}

	origLifecycleHooks := def.LifecycleHooks
	def.LifecycleHooks = []LifecycleHooks{
		combineContainerHooks(defaultHooks, def.LifecycleHooks),
	}

	for _, is := range def.ImageSubstitutors {
		modifiedTag, err := is.Substitute(imageName)
		if err != nil {
			return nil, fmt.Errorf("failed to substitute image %s with %s: %w", imageName, is.Description(), err)
		}

		if modifiedTag != imageName {
			dockerClient.Logger().Info("Replacing image", "description", is.Description(), "from", imageName, "to", modifiedTag)
			imageName = modifiedTag
		}
	}

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
		img, err := dockerClient.Client().ImageInspect(ctx, imageName)
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
		if err := dockerimage.Pull(ctx, dockerClient, imageName, pullOpt); err != nil {
			return nil, err
		}
	}

	// Add the labels that identify this as a container created by the SDK.
	AddSDKLabels(def.Labels)

	dockerInput := &container.Config{
		Entrypoint: def.Entrypoint,
		Image:      imageName,
		Env:        env,
		Labels:     def.Labels,
		Cmd:        def.Cmd,
	}

	hostConfig := &container.HostConfig{}

	networkingConfig := &network.NetworkingConfig{}

	// default hooks include logger hook and pre-create hook
	defaultHooks = append(defaultHooks,
		defaultPreCreateHook(dockerClient, dockerInput, hostConfig, networkingConfig),
		defaultCopyFileToContainerHook(def.Files),
		defaultLogConsumersHook(def.LogConsumerCfg),
		defaultReadinessHook(),
	)

	// Combine with the original LifecycleHooks to avoid duplicate logging hooks.
	def.LifecycleHooks = []LifecycleHooks{
		combineContainerHooks(defaultHooks, origLifecycleHooks),
	}

	err = def.creatingHook(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := dockerClient.Client().ContainerCreate(ctx, dockerInput, hostConfig, networkingConfig, platform, def.Name)
	if err != nil {
		return nil, fmt.Errorf("container create: %w", err)
	}

	// #248: If there is more than one network specified in the request attach newly created container to them one by one
	if len(def.Networks) > 1 {
		for _, n := range def.Networks[1:] {
			nwInspect, err := dockerClient.Client().NetworkInspect(ctx, n, network.InspectOptions{
				Verbose: true,
			})
			if err != nil {
				return nil, fmt.Errorf("network inspect: %w", err)
			}

			endpointSetting := network.EndpointSettings{
				Aliases: def.NetworkAliases[n],
			}
			err = dockerClient.Client().NetworkConnect(ctx, nwInspect.ID, resp.ID, &endpointSetting)
			if err != nil {
				return nil, fmt.Errorf("network connect: %w", err)
			}
		}
	}

	// This should match the fields set in ContainerFromDockerResponse.
	c := &Container{
		dockerClient:   dockerClient,
		ID:             resp.ID,
		shortID:        resp.ID[:12],
		WaitingFor:     def.WaitingFor,
		Image:          imageName,
		exposedPorts:   def.ExposedPorts,
		logger:         dockerClient.Logger(),
		lifecycleHooks: def.LifecycleHooks,
	}

	if err = ctr.createdHook(ctx); err != nil {
		// Return the container to allow caller to clean up.
		return ctr, fmt.Errorf("created hook: %w", err)
	}

	return c, nil
}
