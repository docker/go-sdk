package image_test

import (
	"context"
	"fmt"
	"log"

	apiimage "github.com/docker/docker/api/types/image"
	"github.com/docker/go-sdk/client"
	"github.com/docker/go-sdk/image"
)

func ExamplePull() {
	err := image.Pull(context.Background(), "nginx:latest")

	fmt.Println(err)

	// Output:
	// <nil>
}

func ExamplePull_withClient() {
	dockerClient, err := client.New(context.Background())
	if err != nil {
		log.Printf("error creating client: %s", err)
		return
	}
	defer dockerClient.Close()

	err = image.Pull(context.Background(), "nginx:latest", image.WithPullClient(dockerClient))

	fmt.Println(err)

	// Output:
	// <nil>
}

func ExamplePull_withPullOptions() {
	opts := apiimage.PullOptions{
		Platform: "linux/amd64",
	}

	err := image.Pull(context.Background(), "alpine:3.22", image.WithPullOptions(opts))

	fmt.Println(err)

	// Output:
	// <nil>
}
