package image_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/image"
)

func TestExtractImagesFromDockerfile(t *testing.T) {
	baseImage := "scratch"
	registryHost := "localhost"
	registryPort := "5000"
	nginxImage := "nginx:latest"

	extractImages := func(t *testing.T, dockerfile string, buildArgs map[string]*string, expected []string, expectedError bool) {
		t.Helper()

		images, err := image.ImagesFromDockerfile(dockerfile, buildArgs)
		if expectedError {
			require.Error(t, err)
			require.Empty(t, images)
		} else {
			require.NoError(t, err)
			require.Equal(t, expected, images)
		}
	}

	t.Run("wrong-file", func(t *testing.T) {
		extractImages(t, "", nil, []string{}, true)
	})

	t.Run("single-image", func(t *testing.T) {
		extractImages(t, filepath.Join("testdata", "Dockerfile"), nil, []string{"nginx:${tag}"}, false)
	})

	t.Run("multiple-images", func(t *testing.T) {
		extractImages(t, filepath.Join("testdata", "Dockerfile.multistage"), nil, []string{"nginx:a", "nginx:b", "nginx:c", "scratch"}, false)
	})

	t.Run("multiple-images-with-one-build-arg", func(t *testing.T) {
		extractImages(t, filepath.Join("testdata", "Dockerfile.multistage.singleBuildArgs"), map[string]*string{"BASE_IMAGE": &baseImage}, []string{"nginx:a", "nginx:b", "nginx:c", "scratch"}, false)
	})

	t.Run("multiple-images-with-multiple-build-args", func(t *testing.T) {
		extractImages(t, filepath.Join("testdata", "Dockerfile.multistage.multiBuildArgs"), map[string]*string{"BASE_IMAGE": &baseImage, "REGISTRY_HOST": &registryHost, "REGISTRY_PORT": &registryPort, "NGINX_IMAGE": &nginxImage}, []string{"nginx:latest", "localhost:5000/nginx:latest", "scratch"}, false)
	})
}
