package context_test

import (
	"fmt"
	"log"

	"github.com/docker/go-sdk/context"
)

func ExampleCurrent() {
	ctx, err := context.Current()
	fmt.Println(err)
	fmt.Println(ctx != "")

	// Output:
	// <nil>
	// true
}

func ExampleCurrentDockerHost() {
	host, err := context.CurrentDockerHost()
	fmt.Println(err)
	fmt.Println(host != "")

	// Output:
	// <nil>
	// true
}

func ExampleDockerHostFromContext() {
	host, err := context.DockerHostFromContext("desktop-linux")
	if err != nil {
		log.Printf("error getting docker host from context: %s", err)
		return
	}

	fmt.Println(host)

	// Intentionally not printing the output, as the context could not exist in the CI environment
}
