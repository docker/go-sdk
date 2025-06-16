package container

const (
	version = "0.1.0"
)

// Version returns the version of the docker client.
func Version() string {
	return version
}
