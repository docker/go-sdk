package dockerconfig

import (
	"errors"
	"os"
	"os/exec"
	"testing"
)

// mockExecCommand is a helper function to mock exec.LookPath and exec.Command for testing.
func mockExecCommand(t *testing.T, env ...string) {
	t.Helper()

	execLookPath = func(file string) (string, error) {
		switch file {
		case "docker-credential-helper":
			return os.Args[0], nil
		case "docker-credential-error":
			return "", errors.New("lookup error")
		}

		return "", exec.ErrNotFound
	}

	execCommand = func(name string, arg ...string) *exec.Cmd {
		cmd := exec.Command(name, arg...)
		cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
		cmd.Env = append(cmd.Env, env...)
		return cmd
	}

	t.Cleanup(func() {
		execLookPath = exec.LookPath
		execCommand = exec.Command
	})
}
