package dockercontainer

import (
	"errors"
	"io"
	"log"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-sdk/dockercontainer/wait"
)

// Definition is the Definition of a container.
type Definition struct {
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

	// FromDockerfile embeds the FromDockerfile struct
	FromDockerfile

	// HostConfigModifier the modifier for the host config before container creation
	HostConfigModifier func(*container.HostConfig)

	// HostAccessPorts the ports opened on the host that are accessible to the container.
	HostAccessPorts []int

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

	// Image the image to use for the container.
	Image string

	// ImagePlatform the platform of the image
	ImagePlatform string

	// Name the name of the container.
	Name string

	// AlwaysPullImage whether to always pull the image
	AlwaysPullImage bool

	// Reuse whether to reuse an existing container if it exists or create a new one.
	// A container name must be provided to identify the container to be reused.
	Reuse bool

	// Started whether to auto-start the container.
	Started bool

	// Logger the logger to use for the container.
	Logger log.Logger
}

// File represents a file that will be copied when container starts
type File struct {
	// Reader the reader to read the file from
	Reader io.Reader

	// ContainerPath the path to the file in the container.
	// Use the slash character that matches the path separator of the operating system
	// for the container.
	ContainerPath string

	// Mode the mode of the file
	Mode int64
}

// validate validates the [File]
func (f *File) validate() error {
	if f.Reader == nil {
		return errors.New("reader must be specified")
	}

	if f.ContainerPath == "" {
		return errors.New("container path must be specified")
	}

	return nil
}
