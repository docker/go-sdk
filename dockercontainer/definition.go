package dockercontainer

import (
	"errors"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-sdk/dockerclient"
	"github.com/docker/go-sdk/dockercontainer/wait"
)

// Definition is the Definition of a container.
type Definition struct {
	// DockerClient the docker client to use for the container.
	DockerClient *dockerclient.Client

	// ConfigModifier the modifier for the config before container creation
	ConfigModifier func(*container.Config)

	// Cmd the command to use for the container.
	Cmd []string

	// EndpointSettingsModifier the modifier for the network settings before container creation
	EndpointSettingsModifier func(map[string]*network.EndpointSettings)

	// Entrypoint the entrypoint to use for the container.
	Entrypoint []string

	// Env the environment variables to use for the container.
	Env map[string]string

	// Files the files to be copied when container starts
	Files []File

	// HostConfigModifier the modifier for the host config before container creation
	HostConfigModifier func(*container.HostConfig)

	// ImageSubstitutors the image substitutors to use for the container.
	ImageSubstitutors []ImageSubstitutor

	// Labels the labels to use for the container.
	Labels map[string]string

	// LifecycleHooks the hooks to be executed during container lifecycle
	LifecycleHooks []LifecycleHooks

	// LogConsumerCfg the configuration for the log producer and its log consumers to follow the logs
	LogConsumerCfg *LogConsumerConfig

	// NetworkAliases the network aliases to use for the container.
	NetworkAliases map[string][]string

	// Networks the networks to use for the container.
	Networks []string

	// WaitingFor the waiting strategy to use for the container.
	WaitingFor wait.Strategy

	// 16-byte aligned fields (strings)
	// ExposedPorts the ports exposed by the container.
	ExposedPorts []string

	// image the image to use for the container.
	image string

	// ImagePlatform the platform of the image
	ImagePlatform string

	// Name the name of the container.
	Name string

	// AlwaysPullImage whether to always pull the image
	AlwaysPullImage bool

	// Started whether to auto-start the container.
	Started bool
}

// validate validates the definition.
func (d *Definition) validate() error {
	if d.image == "" {
		return errors.New("image is required")
	}

	return nil
}
