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
	// dockerClient the docker client to use for the container.
	dockerClient *dockerclient.Client

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

	// imageSubstitutors the image substitutors to use for the container.
	imageSubstitutors []ImageSubstitutor

	// labels the labels to use for the container.
	labels map[string]string

	// lifecycleHooks the hooks to be executed during container lifecycle
	lifecycleHooks []LifecycleHooks

	// logConsumerCfg the configuration for the log producer and its log consumers to follow the logs
	logConsumerCfg *LogConsumerConfig

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

	// name the name of the container.
	name string

	// alwaysPullImage whether to always pull the image
	alwaysPullImage bool

	// started whether to auto-start the container.
	started bool
}

// validate validates the definition.
func (d *Definition) validate() error {
	if d.image == "" {
		return errors.New("image is required")
	}

	return nil
}

// DockerClient returns the docker client used by the definition.
func (d *Definition) DockerClient() *dockerclient.Client {
	return d.dockerClient
}

// Image returns the image used by the definition.
func (d *Definition) Image() string {
	return d.image
}
