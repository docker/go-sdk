package context

import (
	"errors"
	"os"
	"path/filepath"
)

var (
	ErrRootlessDockerNotFoundXDGRuntimeDir = errors.New("docker.sock not found in $XDG_RUNTIME_DIR")
	ErrXDGRuntimeDirNotSet                 = errors.New("$XDG_RUNTIME_DIR is not set")
	ErrInvalidSchema                       = errors.New("URL schema is not " + DefaultSchema + " or tcp")
)

// rootlessSocketPathFromEnv returns the path to the rootless Docker socket from the XDG_RUNTIME_DIR environment variable.
// It should include the Docker socket schema (unix://, npipe:// or tcp://) in the returned path.
func rootlessSocketPathFromEnv() (string, error) {
	xdgRuntimeDir, exists := os.LookupEnv("XDG_RUNTIME_DIR")
	if exists && xdgRuntimeDir != "" {
		f := filepath.Join(xdgRuntimeDir, "docker.sock")
		if fileExists(f) {
			return DefaultSchema + f, nil
		}

		return "", ErrRootlessDockerNotFoundXDGRuntimeDir
	}

	return "", ErrXDGRuntimeDirNotSet
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}

	return true
}
