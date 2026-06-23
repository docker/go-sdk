package config

import (
	"errors"
	"os/exec"
	"path/filepath"
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

// setupNonExistentConfigDir points DOCKER_CONFIG at a path that does not exist,
// exercising the hard-fail code path in Dir() when an explicitly overridden
// config directory is missing. Unlike the default ~/.docker case, an explicit
// DOCKER_CONFIG pointing at a non-existent path is a user error and must NOT
// surface ErrConfigFileNotFound.
func setupNonExistentConfigDir(t *testing.T) {
	t.Helper()
	t.Setenv(EnvOverrideDir, filepath.Join(t.TempDir(), "does-not-exist"))
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

func TestDir_OverriddenConfigDirNotFound_NoSentinel(t *testing.T) {
	setupNonExistentConfigDir(t)

	_, err := Dir()
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrConfigFileNotFound)
	require.Contains(t, err.Error(), "file does not exist")
}

func TestDir_DefaultConfigDirNotFound_ReturnsSentinel(t *testing.T) {
	// Point HOME at a temp dir without a .docker subdirectory, and clear
	// DOCKER_CONFIG so Dir() falls back to the default ~/.docker path.
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome) // Windows support
	t.Setenv(EnvOverrideDir, "")

	_, err := Dir()
	require.ErrorIs(t, err, ErrConfigFileNotFound)
	require.Contains(t, err.Error(), "file does not exist")
}

func TestFilepath_OverriddenConfigDirNotFound_NoSentinel(t *testing.T) {
	setupNonExistentConfigDir(t)

	_, err := Filepath()
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrConfigFileNotFound)
}

func TestLoad_OverriddenConfigDirNotFound_NoSentinel(t *testing.T) {
	setupNonExistentConfigDir(t)

	_, err := Load()
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrConfigFileNotFound)
}
