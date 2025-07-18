package container

import (
	"fmt"
	"net/url"

	"github.com/docker/go-sdk/config/auth"
)

// ImageSubstitutor represents a way to substitute container image names
type ImageSubstitutor interface {
	// Description returns the name of the type and a short description of how it modifies the image.
	// Useful to be printed in logs
	Description() string
	Substitute(image string) (string, error)
}

// CustomHubSubstitutor represents a way to substitute the hub of an image with a custom one,
// using provided value with respect to the HubImageNamePrefix configuration value.
type CustomHubSubstitutor struct {
	hub string
}

// NewCustomHubSubstitutor creates a new CustomHubSubstitutor
func NewCustomHubSubstitutor(hub string) CustomHubSubstitutor {
	return CustomHubSubstitutor{
		hub: hub,
	}
}

// Description returns the name of the type and a short description of how it modifies the image.
func (c CustomHubSubstitutor) Description() string {
	return fmt.Sprintf("CustomHubSubstitutor (replaces hub with %s)", c.hub)
}

// Substitute replaces the hub of the image with the provided one, with certain conditions:
//   - if the hub is empty, the image is returned as is.
//   - if the image already contains a registry, the image is returned as is.
func (c CustomHubSubstitutor) Substitute(image string) (string, error) {
	ref, err := auth.ParseImageRef(image)
	if err != nil {
		return "", err
	}

	registry := ref.Registry

	exclusions := []func() bool{
		func() bool { return c.hub == "" },
		func() bool { return registry != auth.DockerRegistry },
	}

	for _, exclusion := range exclusions {
		if exclusion() {
			return image, nil
		}
	}

	result, err := url.JoinPath(c.hub, image)
	if err != nil {
		return "", err
	}

	return result, nil
}

// prependHubRegistry represents a way to prepend a custom Hub registry to the image name,
// using the HubImageNamePrefix configuration value
type prependHubRegistry struct {
	prefix string
}

// newPrependHubRegistry creates a new prependHubRegistry
func newPrependHubRegistry(hubPrefix string) prependHubRegistry {
	return prependHubRegistry{
		prefix: hubPrefix,
	}
}

// Description returns the name of the type and a short description of how it modifies the image.
func (p prependHubRegistry) Description() string {
	return fmt.Sprintf("HubImageSubstitutor (prepends %s)", p.prefix)
}

// Substitute prepends the Hub prefix to the image name, with certain conditions:
//   - if the prefix is empty, the image is returned as is.
//   - if the image is a non-hub image (e.g. where another registry is set), the image is returned as is.
//   - if the image is a Docker Hub image where the hub registry is explicitly part of the name
//     (i.e. anything with a registry.hub.docker.com host part), the image is returned as is.
func (p prependHubRegistry) Substitute(image string) (string, error) {
	ref, err := auth.ParseImageRef(image)
	if err != nil {
		return "", err
	}

	registry := ref.Registry

	// add the exclusions in the right order
	exclusions := []func() bool{
		func() bool { return p.prefix == "" },                  // no prefix set at the configuration level
		func() bool { return registry != auth.DockerRegistry }, // explicitly including Docker's URLs
	}

	for _, exclusion := range exclusions {
		if exclusion() {
			return image, nil
		}
	}

	result, err := url.JoinPath(p.prefix, image)
	if err != nil {
		return "", err
	}

	return result, nil
}
