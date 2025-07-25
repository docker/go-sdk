//go:build !windows
// +build !windows

package context

func init() {
	// DefaultSchema is the default schema to use for the Docker host on Linux
	DefaultSchema = "unix://"

	// DefaultDockerHost is the default host to connect to the Docker socket on Linux
	DefaultDockerHost = DefaultSchema + "/var/run/docker.sock"
}
