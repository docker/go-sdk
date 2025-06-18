package config_test

import (
	"fmt"
	"log"

	"github.com/docker/go-sdk/config"
)

func ExampleRegistryCredentials() {
	authConfig, err := config.RegistryCredentials("nginx:latest")
	if err != nil {
		log.Printf("error getting registry credentials: %s", err)
		return
	}

	fmt.Println(authConfig.ServerAddress)
	fmt.Println(authConfig.Username != "")

	// Output:
	// https://index.docker.io/v1/
	// true
}

func ExampleRegistryCredentialsForHostname() {
	authConfig, err := config.RegistryCredentialsForHostname("https://index.docker.io/v1/")
	if err != nil {
		log.Printf("error getting registry credentials: %s", err)
		return
	}

	fmt.Println(authConfig.ServerAddress)
	fmt.Println(authConfig.Username != "")

	// Output:
	// https://index.docker.io/v1/
	// true
}
