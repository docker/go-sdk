package auth_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/config/auth"
)

const (
	wrongDigest256 = "sha256:123456"
	wrongDigest512 = "sha512:123456"
	testDigest256  = "sha256:7d0d8fa9b6cbbfd96b1a0f0c5e9d5c5f5c5f5c5f5c5f5c5f5c5f5c5f5c5f5c5f"
	testDigest512  = "sha512:7d0d8fa9b6cbbfd96b1a0f0c5e9d5c5f5c5f5c5f5c5f5c5f5c5f5c5f5c5f5c5f7d0d8fa9b6cbbfd96b1a0f0c5e9d5c5f5c5f5c5f5c5f5c5f5c5f5c5f5c5f5c5f"
)

func TestParseImageRef(t *testing.T) {
	t.Run("empty-image", func(t *testing.T) {
		ref, err := auth.ParseImageRef("")
		require.Error(t, err)
		require.Empty(t, ref)
	})

	t.Run("numbers", func(t *testing.T) {
		ref, err := auth.ParseImageRef("1234567890")
		require.NoError(t, err)
		require.Equal(t, auth.DockerRegistry, ref.Registry)
		require.Equal(t, "library/1234567890", ref.Repository)
		require.Empty(t, ref.Tag)
		require.Empty(t, ref.Digest)
	})

	t.Run("malformed-image", func(t *testing.T) {
		ref, err := auth.ParseImageRef("--malformed--")
		require.Error(t, err)
		require.Empty(t, ref)
	})

	t.Run("protocol-registry", func(t *testing.T) {
		ref, err := auth.ParseImageRef("http://docker.io/library/nginx")
		require.Error(t, err)
		require.Empty(t, ref)
	})

	suiteParseImageRefFn := func(t *testing.T, testRegistry string) {
		t.Helper()

		expectedRegistry := testRegistry
		expectedRepositoryPrefix := ""
		if testRegistry != "" {
			testRegistry += "/"
		} else {
			expectedRepositoryPrefix = "library/"
			expectedRegistry = auth.DockerRegistry
		}

		t.Run("image", func(t *testing.T) {
			ref, err := auth.ParseImageRef(testRegistry + "nginx")
			require.NoError(t, err)
			require.Equal(t, expectedRegistry, ref.Registry)
			require.Equal(t, expectedRepositoryPrefix+"nginx", ref.Repository)
			require.Empty(t, ref.Tag)
			require.Empty(t, ref.Digest)
		})

		t.Run("image@256digest", func(t *testing.T) {
			ref, err := auth.ParseImageRef(testRegistry + "nginx@" + string(testDigest256))
			require.NoError(t, err)
			require.Equal(t, expectedRegistry, ref.Registry)
			require.Equal(t, expectedRepositoryPrefix+"nginx", ref.Repository)
			require.Empty(t, ref.Tag)
			require.Equal(t, testDigest256, ref.Digest)
		})

		t.Run("image@512digest", func(t *testing.T) {
			ref, err := auth.ParseImageRef(testRegistry + "nginx@" + string(testDigest512))
			require.NoError(t, err)
			require.Equal(t, expectedRegistry, ref.Registry)
			require.Equal(t, expectedRepositoryPrefix+"nginx", ref.Repository)
			require.Empty(t, ref.Tag)
			require.Equal(t, testDigest512, ref.Digest)
		})

		t.Run("image:tag", func(t *testing.T) {
			ref, err := auth.ParseImageRef(testRegistry + "nginx:latest")
			require.NoError(t, err)
			require.Equal(t, expectedRegistry, ref.Registry)
			require.Equal(t, expectedRepositoryPrefix+"nginx", ref.Repository)
			require.Equal(t, "latest", ref.Tag)
			require.Empty(t, ref.Digest)
		})

		t.Run("image:tag@256digest", func(t *testing.T) {
			ref, err := auth.ParseImageRef(testRegistry + "nginx:latest@" + string(testDigest256))
			require.NoError(t, err)
			require.Equal(t, expectedRegistry, ref.Registry)
			require.Equal(t, expectedRepositoryPrefix+"nginx", ref.Repository)
			require.Equal(t, "latest", ref.Tag)
			require.Equal(t, testDigest256, ref.Digest)
		})

		t.Run("image:tag@512digest", func(t *testing.T) {
			ref, err := auth.ParseImageRef(testRegistry + "nginx:latest@" + string(testDigest512))
			require.NoError(t, err)
			require.Equal(t, expectedRegistry, ref.Registry)
			require.Equal(t, expectedRepositoryPrefix+"nginx", ref.Repository)
			require.Equal(t, "latest", ref.Tag)
			require.Equal(t, testDigest512, ref.Digest)
		})

		t.Run("repository/image", func(t *testing.T) {
			ref, err := auth.ParseImageRef(testRegistry + "testcontainers/ryuk")
			require.NoError(t, err)
			require.Equal(t, expectedRegistry, ref.Registry)
			require.Equal(t, "testcontainers/ryuk", ref.Repository)
			require.Empty(t, ref.Tag)
			require.Empty(t, ref.Digest)
		})

		t.Run("repository/image@256digest", func(t *testing.T) {
			ref, err := auth.ParseImageRef(testRegistry + "testcontainers/ryuk@" + string(testDigest256))
			require.NoError(t, err)
			require.Equal(t, expectedRegistry, ref.Registry)
			require.Equal(t, "testcontainers/ryuk", ref.Repository)
			require.Empty(t, ref.Tag)
			require.Equal(t, testDigest256, ref.Digest)
		})

		t.Run("repository/image@512digest", func(t *testing.T) {
			ref, err := auth.ParseImageRef(testRegistry + "testcontainers/ryuk@" + string(testDigest512))
			require.NoError(t, err)
			require.Equal(t, expectedRegistry, ref.Registry)
			require.Equal(t, "testcontainers/ryuk", ref.Repository)
			require.Empty(t, ref.Tag)
			require.Equal(t, testDigest512, ref.Digest)
		})

		t.Run("repository/image:tag", func(t *testing.T) {
			ref, err := auth.ParseImageRef(testRegistry + "testcontainers/ryuk:latest")
			require.NoError(t, err)
			require.Equal(t, expectedRegistry, ref.Registry)
			require.Equal(t, "testcontainers/ryuk", ref.Repository)
			require.Equal(t, "latest", ref.Tag)
			require.Empty(t, ref.Digest)
		})

		t.Run("repository/image:tag@256digest", func(t *testing.T) {
			ref, err := auth.ParseImageRef(testRegistry + "testcontainers/ryuk:latest@" + string(testDigest256))
			require.NoError(t, err)
			require.Equal(t, expectedRegistry, ref.Registry)
			require.Equal(t, "testcontainers/ryuk", ref.Repository)
			require.Equal(t, "latest", ref.Tag)
			require.Equal(t, testDigest256, ref.Digest)
		})

		t.Run("repository/image:tag@wrong-256digest", func(t *testing.T) {
			ref, err := auth.ParseImageRef(testRegistry + "testcontainers/ryuk:latest@" + string(wrongDigest256))
			require.Error(t, err)
			require.Empty(t, ref)
		})

		t.Run("repository/image:tag@512digest", func(t *testing.T) {
			ref, err := auth.ParseImageRef(testRegistry + "testcontainers/ryuk:latest@" + string(testDigest512))
			require.NoError(t, err)
			require.Equal(t, expectedRegistry, ref.Registry)
			require.Equal(t, "testcontainers/ryuk", ref.Repository)
			require.Equal(t, "latest", ref.Tag)
			require.Equal(t, testDigest512, ref.Digest)
		})

		t.Run("repository/image:tag@wrong-512digest", func(t *testing.T) {
			ref, err := auth.ParseImageRef(testRegistry + "testcontainers/ryuk:latest@" + string(wrongDigest512))
			require.Error(t, err)
			require.Empty(t, ref)
		})
	}

	t.Run("no-registry", func(t *testing.T) {
		suiteParseImageRefFn(t, "")
	})

	t.Run("localhost-registry/port", func(t *testing.T) {
		suiteParseImageRefFn(t, "localhost:5000")
	})

	t.Run("host-registry/port", func(t *testing.T) {
		suiteParseImageRefFn(t, "server.internal:5000")
	})

	t.Run("ip-registry/port", func(t *testing.T) {
		suiteParseImageRefFn(t, "127.0.0.1:5000")
	})

	t.Run("dns-registry", func(t *testing.T) {
		suiteParseImageRefFn(t, "docker.elastic.co")
	})
}
