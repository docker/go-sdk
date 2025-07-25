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

func ExampleNew() {
	ctx, err := context.New("my-context")
	if err != nil {
		log.Printf("error adding context: %s", err)
		return
	}
	defer func() {
		if err := ctx.Delete(); err != nil {
			log.Printf("error deleting context: %s", err)
		}
	}()

	fmt.Println(ctx.Name)

	// Output:
	// my-context
}

func ExampleNew_asCurrent() {
	ctx, err := context.New("my-context", context.AsCurrent(), context.WithHost("tcp://127.0.0.1:2375"))
	if err != nil {
		log.Printf("error adding context: %s", err)
		return
	}
	defer func() {
		if err := ctx.Delete(); err != nil {
			log.Printf("error deleting context: %s", err)
		}
	}()

	fmt.Println(ctx.Name)

	current, err := context.Current()
	if err != nil {
		log.Printf("error getting current context: %s", err)
		return
	}
	fmt.Println(current)

	host, err := context.CurrentDockerHost()
	if err != nil {
		log.Printf("error getting current docker host: %s", err)
		return
	}

	fmt.Println(host)

	// Output:
	// my-context
	// my-context
	// tcp://127.0.0.1:2375
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

	fmt.Println(ctx.Metadata.Description)
	fmt.Println(ctx.Metadata.Field("otel"))
	fmt.Println(ctx.Metadata.Fields())

	// Intentionally not printing the output, as the context could not exist in the CI environment
}
