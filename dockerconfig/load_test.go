package dockerconfig

import (
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed testdata/.docker/config.json
var dockerConfig string

func TestLoad(t *testing.T) {
	var expectedConfig Config
	err := json.Unmarshal([]byte(dockerConfig), &expectedConfig)
	require.NoError(t, err)

	t.Run("HOME", func(t *testing.T) {
		t.Run("valid", func(t *testing.T) {
			setupHome(t, "testdata")

			cfg, err := Load()
			require.NoError(t, err)
			require.Equal(t, expectedConfig, cfg)
		})

		t.Run("not-found", func(t *testing.T) {
			setupHome(t, "testdata", "not-found")

			cfg, err := Load()
			require.ErrorIs(t, err, os.ErrNotExist)
			require.Empty(t, cfg)
		})

		t.Run("invalid-config", func(t *testing.T) {
			setupHome(t, "testdata", "invalid-config")

			cfg, err := Load()
			require.ErrorContains(t, err, "json: cannot unmarshal array")
			require.Empty(t, cfg)
		})
	})

	t.Run("DOCKER_AUTH_CONFIG", func(t *testing.T) {
		t.Run("valid", func(t *testing.T) {
			setupHome(t, "testdata", "not-found")
			t.Setenv("DOCKER_AUTH_CONFIG", dockerConfig)

			cfg, err := Load()
			require.NoError(t, err)
			require.Equal(t, expectedConfig, cfg)
		})

		t.Run("invalid-config", func(t *testing.T) {
			setupHome(t, "testdata", "not-found")
			t.Setenv("DOCKER_AUTH_CONFIG", `{"auths": []}`)

			cfg, err := Load()
			require.ErrorContains(t, err, "json: cannot unmarshal array")
			require.Empty(t, cfg)
		})
	})

	t.Run(EnvOverrideDir, func(t *testing.T) {
		t.Run("valid", func(t *testing.T) {
			setupHome(t, "testdata", "not-found")
			t.Setenv(EnvOverrideDir, filepath.Join("testdata", ".docker"))

			cfg, err := Load()
			require.NoError(t, err)
			require.Equal(t, expectedConfig, cfg)
		})

		t.Run("invalid-config", func(t *testing.T) {
			setupHome(t, "testdata", "not-found")
			t.Setenv(EnvOverrideDir, filepath.Join("testdata", "invalid-config", ".docker"))

			cfg, err := Load()
			require.ErrorContains(t, err, "json: cannot unmarshal array")
			require.Empty(t, cfg)
		})
	})
}

func TestDir(t *testing.T) {
	t.Run("HOME", func(t *testing.T) {
		t.Run("valid", func(t *testing.T) {
			tmpDir := t.TempDir()
			setupHome(t, tmpDir)

			// create the Docker config directory
			cfgDir := filepath.Join(tmpDir, configFileDir)
			err := os.Mkdir(cfgDir, 0o755)
			require.NoError(t, err)

			dir, err := Dir()
			require.NoError(t, err)
			require.Equal(t, cfgDir, dir)
		})

		t.Run("not-found", func(t *testing.T) {
			setupHome(t, "testdata", "not-found")

			dir, err := Dir()
			require.ErrorIs(t, err, os.ErrNotExist)
			require.Empty(t, dir)
		})
	})

	t.Run(EnvOverrideDir, func(t *testing.T) {
		t.Run("valid", func(t *testing.T) {
			tmpDir := t.TempDir()
			setupDockerConfigs(t, tmpDir)

			dir, err := Dir()
			require.NoError(t, err)
			require.Equal(t, tmpDir, dir)
		})

		t.Run("not-found", func(t *testing.T) {
			setupDockerConfigs(t, "testdata", "not-found")

			dir, err := Dir()
			require.ErrorIs(t, err, os.ErrNotExist)
			require.Empty(t, dir)
		})
	})
}

// setupHome sets the user's home directory to the given path
// It also creates the Docker config directory.
func setupHome(t *testing.T, dirs ...string) {
	t.Helper()

	dir := filepath.Join(dirs...)
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir) // Windows
}

// setupDockerConfigs sets the DOCKER_CONFIG environment variable to the given path,
// and the DOCKER_AUTH_CONFIG environment variable to the testdata/dockerconfig/config.json file.
func setupDockerConfigs(t *testing.T, dirs ...string) {
	t.Helper()

	dir := filepath.Join(dirs...)
	t.Setenv("DOCKER_AUTH_CONFIG", dockerConfig)
	t.Setenv(EnvOverrideDir, dir)
}
