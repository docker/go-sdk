//go:build !windows
// +build !windows

package context

func init() {
	// DefaultDockerHost is the default host to connect to the Docker socket on Linux
	DefaultDockerHost = "unix:///var/run/docker.sock"
}
