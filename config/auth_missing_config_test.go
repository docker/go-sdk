package config

import (
	"errors"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

// setupConfigDirWithoutFile creates a temporary directory and sets DOCKER_CONFIG
// to point to it, but does NOT create a config.json file. This exercises the
// ErrConfigFileNotFound code path in Load/Filepath.
func setupConfigDirWithoutFile(t *testing.T) {
	t.Helper()
	t.Setenv(EnvOverrideDir, t.TempDir())
}

func TestAuthConfigs_ConfigNotFound(t *testing.T) {
	setupConfigDirWithoutFile(t)
	mockExecCommand(t)

	authConfigs, err := AuthConfigs("some.io/repo/image:tag")
	require.NoError(t, err)
	require.Contains(t, authConfigs, "some.io")
	require.Empty(t, authConfigs["some.io"].Username)
	require.Empty(t, authConfigs["some.io"].Password)
}

func TestAuthConfigs_ConfigNotFound_FallsBackToCredentialHelper(t *testing.T) {
	setupConfigDirWithoutFile(t)

	execLookPath = func(string) (string, error) {
		return "", errors.New("helper unreachable")
	}
	t.Cleanup(func() { execLookPath = exec.LookPath })

	_, err := AuthConfigs("some.io/repo/image:tag")
	require.Error(t, err)
	require.ErrorContains(t, err, "helper unreachable")
}

func TestAuthConfigForHostname_ConfigNotFound(t *testing.T) {
	setupConfigDirWithoutFile(t)
	mockExecCommand(t)

	creds, err := AuthConfigForHostname("some.io")
	require.NoError(t, err)
	require.Empty(t, creds.Username)
	require.Empty(t, creds.Password)
}

func TestAuthConfigForHostname_ConfigNotFound_FallsBackToCredentialHelper(t *testing.T) {
	setupConfigDirWithoutFile(t)

	execLookPath = func(string) (string, error) {
		return "", errors.New("helper unreachable")
	}
	t.Cleanup(func() { execLookPath = exec.LookPath })

	_, err := AuthConfigForHostname("some.io")
	require.Error(t, err)
	require.ErrorContains(t, err, "helper unreachable")
}

func TestLoad_ConfigNotFound_ReturnsSentinel(t *testing.T) {
	setupConfigDirWithoutFile(t)

	_, err := Load()
	require.ErrorIs(t, err, ErrConfigFileNotFound)
}

func TestFilepath_ConfigNotFound_ReturnsSentinel(t *testing.T) {
	setupConfigDirWithoutFile(t)

	_, err := Filepath()
	require.ErrorIs(t, err, ErrConfigFileNotFound)
	require.Contains(t, err.Error(), "config file does not exist")
}
