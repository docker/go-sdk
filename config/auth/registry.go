package auth

import (
	"fmt"
	"strings"

	"github.com/distribution/reference"
)

const (
	IndexDockerIO  = "https://index.docker.io/v1/"
	DockerRegistry = "docker.io"
)

// ImageReference represents a parsed Docker image reference
type ImageReference struct {
	// Registry is the registry hostname (e.g., "docker.io", "myregistry.com:5000")
	Registry string
	// Repository is the image repository (e.g., "library/nginx", "user/image")
	Repository string
	// Tag is the image tag (e.g., "latest", "v1.0.0")
	Tag string
	// Digest is the image digest if present (e.g., "sha256:...")
	Digest string
}

// ParseImageRef extracts the registry from the image name, using github.com/distribution/reference as a reference parser,
// and returns the ImageReference struct.
func ParseImageRef(imageRef string) (ImageReference, error) {
	ref, err := reference.ParseAnyReference(imageRef)
	if err != nil {
		return ImageReference{}, fmt.Errorf("parse image ref: %w", err)
	}

	imgRef := ImageReference{}

	named, namedOk := ref.(reference.Named)
	if namedOk {
		imgRef.Registry = reference.Domain(named)
		imgRef.Repository = reference.Path(named)
	}

	tagged, ok := ref.(reference.Tagged)
	if ok {
		imgRef.Tag = tagged.Tag()
	}

	digest, ok := ref.(reference.Digested)
	if ok {
		imgRef.Digest = string(digest.Digest())
	}

	return imgRef, nil
}

// resolveRegistryHost can be used to transform a docker registry host name into what is used for the docker config/cred helpers
//
// This is useful for using with containerd authorizers.
// Naturally this only transforms docker hub URLs.
func resolveRegistryHost(host string) string {
	// Strip http:// and https:// prefixes if present
	if strings.HasPrefix(host, "https://") {
		host = strings.TrimPrefix(host, "https://")
	} else if strings.HasPrefix(host, "http://") {
		host = strings.TrimPrefix(host, "http://")
	}

	switch host {
	case "index.docker.io", "docker.io", "index.docker.io/v1/", "registry-1.docker.io":
		return IndexDockerIO
	}
	return host
}
