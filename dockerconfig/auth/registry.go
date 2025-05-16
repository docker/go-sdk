package auth

import (
	"fmt"
	"regexp"
)

const (
	IndexDockerIO = "https://index.docker.io/v1/"

	// Protocol part (optional)
	protocolGroup = `(?:https?://)?`

	// Hostname part (domain, IP, or localhost)
	hostnameGroup = `(?:(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}|(?:\d{1,3}\.){3}\d{1,3}|localhost)`

	// Port part (optional)
	portGroup = `(?::\d+)?`

	// Registry part (must be a valid hostname/IP with optional protocol and port)
	registryGroup = `(?:(?P<registry>` + protocolGroup + hostnameGroup + portGroup + `)/)?`

	// Repository part (can be single or multi-level)
	repositoryGroup = `(?P<repository>(?:[^/:@]+/)*[^/:@]+)`

	// Tag part
	tagGroup = `(?::(?P<tag>[^@]+))?`

	// Digest part
	digestGroup = `(?:@(?P<digest>sha256:[a-f0-9]{64}|sha512:[a-f0-9]{128}))?`

	// 1. registry/repository[tag][digest]
	// 2. repository[tag][digest] (when no registry)
	regexImageRef = `^` + registryGroup + repositoryGroup + tagGroup + digestGroup + `$`
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

// ParseImageRef extracts the registry from the image name, using a regular expression to extract the registry from the image name.
// - image:tag
// - image:tag@digest
// - image
// - image@digest
// - repository/image:tag
// - repository/image:tag@digest
// - repository/image
// - repository/image@digest
// - registry/image:tag
// - registry/image:tag@digest
// - registry/image
// - registry/image@digest
// - registry/repository/image:tag
// - registry/repository/image:tag@digest
// - registry/repository/image
// - registry/repository/image@digest
// - registry:port/repository/image:tag
// - registry:port/repository/image:tag@digest
// - registry:port/repository/image
// - registry:port/repository/image@digest
// - registry:port/image:tag
// - registry:port/image:tag@digest
// - registry:port/image
// - registry:port/image@digest
// Once extracted the registry, it is validated to return the Docker Index URL
// if the registry is a valid Docker Index URL, otherwise it returns the registry as is.
func ParseImageRef(imageRef string) (ImageReference, error) {
	var ref ImageReference

	r := regexp.MustCompile(regexImageRef)

	matches := r.FindStringSubmatch(imageRef)
	if len(matches) == 0 {
		return ref, fmt.Errorf("invalid image reference: %s", imageRef)
	}

	// Get named groups
	names := r.SubexpNames()
	result := make(map[string]string)
	for i, name := range names {
		if i != 0 && name != "" { // Skip the first empty name
			result[name] = matches[i]
		}
	}

	if result["registry"] == "" {
		result["registry"] = IndexDockerIO
	}

	ref = ImageReference{
		Registry:   resolveRegistryHost(result["registry"]),
		Repository: result["repository"],
		Tag:        result["tag"],
		Digest:     result["digest"],
	}

	return ref, nil
}

// resolveRegistryHost can be used to transform a docker registry host name into what is used for the docker config/cred helpers
//
// This is useful for using with containerd authorizers.
// Naturally this only transforms docker hub URLs.
func resolveRegistryHost(host string) string {
	switch host {
	case "index.docker.io", "docker.io", IndexDockerIO, "registry-1.docker.io":
		return IndexDockerIO
	}
	return host
}
