package container

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImageSubstitutors(t *testing.T) {
	t.Run("custom-hub", func(t *testing.T) {
		t.Run("prepend-registry", func(t *testing.T) {
			s := NewCustomHubSubstitutor("quay.io")

			img, err := s.Substitute("foo/foo:latest")
			require.NoError(t, err)

			require.Equal(t, "quay.io/foo/foo:latest", img)
		})

		t.Run("no-prepend-same-registry", func(t *testing.T) {
			s := NewCustomHubSubstitutor("quay.io")

			img, err := s.Substitute("quay.io/foo/foo:latest")
			require.NoError(t, err)

			require.Equal(t, "quay.io/foo/foo:latest", img)
		})
	})

	t.Run("docker-hub", func(t *testing.T) {
		t.Run("prepend-registry", func(t *testing.T) {
			t.Run("image", func(t *testing.T) {
				s := newPrependHubRegistry("my-registry")

				img, err := s.Substitute("foo:latest")
				require.NoError(t, err)

				require.Equal(t, "my-registry/foo:latest", img)
			})
			t.Run("image/user", func(t *testing.T) {
				s := newPrependHubRegistry("my-registry")

				img, err := s.Substitute("user/foo:latest")
				require.NoError(t, err)

				require.Equal(t, "my-registry/user/foo:latest", img)
			})

			t.Run("image/organization/user", func(t *testing.T) {
				s := newPrependHubRegistry("my-registry")

				img, err := s.Substitute("org/user/foo:latest")
				require.NoError(t, err)

				require.Equal(t, "my-registry/org/user/foo:latest", img)
			})
		})

		t.Run("no-prepend-registry", func(t *testing.T) {
			t.Run("non-hub-image", func(t *testing.T) {
				s := newPrependHubRegistry("my-registry")

				img, err := s.Substitute("quay.io/foo:latest")
				require.NoError(t, err)

				require.Equal(t, "quay.io/foo:latest", img)
			})

			t.Run("registry.hub.docker.com/library", func(t *testing.T) {
				s := newPrependHubRegistry("my-registry")

				img, err := s.Substitute("registry.hub.docker.com/library/foo:latest")
				require.NoError(t, err)

				require.Equal(t, "registry.hub.docker.com/library/foo:latest", img)
			})

			t.Run("registry.hub.docker.com", func(t *testing.T) {
				s := newPrependHubRegistry("my-registry")

				img, err := s.Substitute("registry.hub.docker.com/foo:latest")
				require.NoError(t, err)

				require.Equal(t, "registry.hub.docker.com/foo:latest", img)
			})
		})
	})
}
