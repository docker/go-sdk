package container

import (
	"errors"
	"fmt"
	"strings"

	"github.com/containerd/platforms"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-sdk/client"
	"github.com/docker/go-sdk/container/wait"
)

var (
	// ErrDuplicateMountTarget is returned when a duplicate mount target is detected.
	ErrDuplicateMountTarget = errors.New("duplicate mount target detected")

	// ErrInvalidBindMount is returned when an invalid bind mount is detected.
	ErrInvalidBindMount = errors.New("invalid bind mount")
)

// Definition is the definition of a container.
type Definition struct {
	// dockerClient the docker client to use for the container.
	dockerClient client.SDKClient

	// configModifier the modifier for the config before container creation
	configModifier func(*container.Config)

	// cmd the command to use for the container.
	cmd []string

	// endpointSettingsModifier the modifier for the network settings before container creation
	endpointSettingsModifier func(map[string]*network.EndpointSettings)

	// entrypoint the entrypoint to use for the container.
	entrypoint []string

	// env the environment variables to use for the container.
	env map[string]string

	// files the files to be copied when container starts
	files []File

	// hostConfigModifier the modifier for the host config before container creation
	hostConfigModifier func(*container.HostConfig)

	// validateFuncs the functions to validate the definition.
	validateFuncs []func() error

	// imageSubstitutors the image substitutors to use for the container.
	imageSubstitutors []ImageSubstitutor

	// labels the labels to use for the container.
	labels map[string]string

	// lifecycleHooks the hooks to be executed during container lifecycle
	lifecycleHooks []LifecycleHooks

	// networkAliases the network aliases to use for the container.
	networkAliases map[string][]string

	// networks the networks to use for the container.
	networks []string

	// waitingFor the waiting strategy to use for the container.
	waitingFor wait.Strategy

	// exposedPorts the ports exposed by the container.
	exposedPorts []string

	// image the image to use for the container.
	image string

	// imagePlatform the platform of the image
	imagePlatform string

	// platform the platform of the container.
	// Used to override the platform of the image when building the container.
	platform *platforms.Platform

	// name the name of the container.
	name string

	// alwaysPullImage whether to always pull the image
	alwaysPullImage bool

	// started whether to auto-start the container.
	started bool
}

// validate validates the definition.
func (d *Definition) validate() error {
	var errs []error
	for _, fn := range d.validateFuncs {
		if err := fn(); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// DockerClient returns the docker client used by the definition.
func (d *Definition) DockerClient() client.SDKClient {
	return d.dockerClient
}

// Image returns the image used by the definition.
func (d *Definition) Image() string {
	return d.image
}

// ImageSubstitutors returns the image substitutors used by the definition.
func (d *Definition) ImageSubstitutors() []ImageSubstitutor {
	return d.imageSubstitutors
}

// Labels returns the labels used by the definition.
func (d *Definition) Labels() map[string]string {
	if d.labels == nil {
		d.labels = make(map[string]string)
	}

	return d.labels
}

// Name returns the name of the container.
func (d *Definition) Name() string {
	return d.name
}

// Networks returns the networks of the container.
func (d *Definition) Networks() []string {
	return d.networks
}

// HostConfigModifier returns the host config modifier of the container.
func (d *Definition) HostConfigModifier() func(*container.HostConfig) {
	return d.hostConfigModifier
}

// validateMounts ensures that the mounts do not have duplicate targets.
// It will check the HostConfigModifier.Binds field.
func (d *Definition) validateMounts() error {
	targets := make(map[string]bool, 0)

	if d.hostConfigModifier == nil {
		return nil
	}

	hostConfig := container.HostConfig{}

	d.hostConfigModifier(&hostConfig)

	if len(hostConfig.Binds) > 0 {
		for _, bind := range hostConfig.Binds {
			parts := strings.Split(bind, ":")
			if len(parts) != 2 && len(parts) != 3 {
				return fmt.Errorf("%w: %s", ErrInvalidBindMount, bind)
			}
			targetPath := parts[1]
			if targets[targetPath] {
				return fmt.Errorf("%w: %s", ErrDuplicateMountTarget, targetPath)
			}
			targets[targetPath] = true
		}
	}

	return nil
}
