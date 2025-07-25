//go:build windows
// +build windows

package context

func init() {
	// DefaultSchema is the default schema to use for the Docker host on Windows
	DefaultSchema = "npipe://"

	// DefaultDockerHost is the default host to connect to the Docker socket on Windows
	DefaultDockerHost = DefaultSchema + "//./pipe/docker_engine"
}
