package config_test

import (
	"fmt"

	"github.com/docker/go-sdk/config"
)

func ExampleRegistryCredentials() {
	authConfig, err := config.RegistryCredentials("nginx:latest")
	fmt.Println(err)
	fmt.Println(authConfig.Username != "")

	// Output:
	// <nil>
	// true
}

func ExampleRegistryCredentialsForHostname() {
	authConfig, err := config.RegistryCredentialsForHostname("https://index.docker.io/v1/")
	fmt.Println(err)
	fmt.Println(authConfig.Username != "")

	// Output:
	// <nil>
	// true
}
