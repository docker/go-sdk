package context

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var (
	ErrRootlessDockerNotFoundXDGRuntimeDir = errors.New("$XDG_RUNTIME_DIR does not exist")
	ErrXDGRuntimeDirNotSet                 = errors.New("XDG_RUNTIME_DIR is not set")
	ErrInvalidSchema                       = errors.New("URL schema is not " + DefaultSchema + " or tcp")
)

// rootlessSocketPathFromEnv returns the path to the rootless Docker socket from the XDG_RUNTIME_DIR environment variable.
// It should include the Docker socket schema (unix://) in the returned path.
func rootlessSocketPathFromEnv() (string, error) {
	xdgRuntimeDir, exists := os.LookupEnv("XDG_RUNTIME_DIR")
	if exists {
		f := filepath.Join(xdgRuntimeDir, "docker.sock")
		if err := fileExists(f); err == nil {
			return DefaultSchema + f, nil
		}

		return "", ErrRootlessDockerNotFoundXDGRuntimeDir
	}

	return "", ErrXDGRuntimeDirNotSet
}

// fileExists checks if a file exists.
func fileExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %w", err)
	}

	return nil
}
