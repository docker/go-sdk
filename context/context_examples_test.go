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

func ExampleList() {
	contexts, err := context.List()
	if err != nil {
		log.Printf("error listing contexts: %s", err)
		return
	}

	fmt.Println(contexts)

	// Intentionally not printing the output, as the contexts could not exist in the CI environment
}

func ExampleInspect() {
	ctx, err := context.Inspect("docker-cloud")
	if err != nil {
		log.Printf("error inspecting context: %s", err)
		return
	}

	fmt.Println(ctx.Context.Description)
	fmt.Println(ctx.Context.Field("otel"))
	fmt.Println(ctx.Context.Fields())

	// Intentionally not printing the output, as the context could not exist in the CI environment
}
