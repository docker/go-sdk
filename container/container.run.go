package container

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/docker/api/types/container"
	apinetwork "github.com/docker/docker/api/types/network"
	"github.com/docker/go-sdk/client"
)

// Run is a convenience function that creates a new container and starts it.
// By default, the container is started after creation, unless requested otherwise
// using the [WithNoStart] option.
func Run(ctx context.Context, opts ...ContainerCustomizer) (*Container, error) {
	def := Definition{
		env:     make(map[string]string),
		started: true,
	}

	// initialize the validate functions with the default ones
	def.validateFuncs = []func() error{
		func() error {
			if def.image == "" {
				return errors.New("image is required")
			}
			return nil
		},
		def.validateMounts,
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
		sdk, err := client.New(ctx)
		if err != nil {
			return nil, err
		}
		// use the default docker client
		def.dockerClient = sdk
	}

	env := []string{}
	for envKey, envVar := range def.env {
		env = append(env, envKey+"="+envVar)
	}

	if def.labels == nil {
		def.labels = make(map[string]string)
	}

	defaultHooks := []LifecycleHooks{
		DefaultLoggingHook,
	}

	def.labels[moduleLabel] = Version()

	dockerInput := &container.Config{
		Entrypoint: def.entrypoint,
		Image:      def.image,
		Env:        env,
		Labels:     def.labels, // the sdkClient will add the SDK labels automatically
		Cmd:        def.cmd,
	}

	hostConfig := &container.HostConfig{}

	networkingConfig := &apinetwork.NetworkingConfig{}

	// default hooks include logger hook and pre-create hook
	defaultHooks = append(defaultHooks,
		defaultPreCreateHook(def.dockerClient, dockerInput, hostConfig, networkingConfig),
		defaultCopyFileToContainerHook(def.files),
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

	// Image substitution must be done after the creating hook has been called,
	// as the image could have been overridden in there.
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

	// Update the image name in the docker input after the creating hook has been called,
	// as it could have been overridden in there.
	dockerInput.Image = def.image

	resp, err := def.dockerClient.ContainerCreate(ctx, dockerInput, hostConfig, networkingConfig, def.platform, def.name)
	if err != nil {
		return nil, fmt.Errorf("container create: %w", err)
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

	// Note: `ctr.dockerClient` is the same instance as `def.dockerClient`.
	// The switch is intentional to emphasize that operations are now being performed
	// on the container object (`ctr`) rather than the definition object (`def`).

	// If there is more than one network specified in the request attach newly created container to them one by one
	if len(def.networks) > 1 {
		for _, n := range def.networks[1:] {
			nwInspect, err := ctr.dockerClient.NetworkInspect(ctx, n, apinetwork.InspectOptions{
				Verbose: true,
			})
			if err != nil {
				return ctr, fmt.Errorf("network inspect: %w", err)
			}

			endpointSetting := apinetwork.EndpointSettings{
				Aliases: def.networkAliases[n],
			}
			err = ctr.dockerClient.NetworkConnect(ctx, nwInspect.ID, resp.ID, &endpointSetting)
			if err != nil {
				return ctr, fmt.Errorf("network connect: %w", err)
			}
		}
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
