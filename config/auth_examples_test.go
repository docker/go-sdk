package config_test

import (
	"fmt"

	"github.com/docker/go-sdk/config"
	"github.com/docker/go-sdk/config/auth"
)

func ExampleAuthConfigs() {
	authConfigs, err := config.AuthConfigs("nginx:latest")
	fmt.Println(err)
	fmt.Println(len(authConfigs))
	fmt.Println(authConfigs[auth.IndexDockerIO].Username != "")

	// Output:
	// <nil>
	// 1
	// true
}

func ExampleAuthConfigForHostname() {
	authConfig, err := config.AuthConfigForHostname(auth.IndexDockerIO)
	fmt.Println(err)
	fmt.Println(authConfig.Username != "")

	// Output:
	// <nil>
	// true
}
