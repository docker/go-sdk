package image_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	dockerclient "github.com/moby/moby/client"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"

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
	opts := dockerclient.ImagePullOptions{Platforms: []v1.Platform{
		{
			OS:           "linux",
			Architecture: "amd64",
		},
	}}

	err := image.Pull(context.Background(), "alpine:3.22", image.WithPullOptions(opts))

	fmt.Println(err)

	// Output:
	// <nil>
}

func ExamplePull_withPullHandler() {
	opts := dockerclient.ImagePullOptions{Platforms: []v1.Platform{
		{
			OS:           "linux",
			Architecture: "amd64",
		},
	}}

	buff := &bytes.Buffer{}

	err := image.Pull(context.Background(), "alpine:3.22", image.WithPullOptions(opts), image.WithPullHandler(func(r io.ReadCloser) error {
		_, err := io.Copy(buff, r)
		return err
	}))

	fmt.Println(err)
	fmt.Println(strings.Contains(buff.String(), "Pulling from library/alpine"))

	// Output:
	// <nil>
	// true
}

func ExampleDisplayProgress() {
	// Display formatted pull progress to a custom writer (buffer in this example).
	// Verifies that DisplayProgress formats the output (not raw JSON).
	buff := &bytes.Buffer{}

	err := image.Pull(context.Background(), "nginx:latest",
		image.WithPullHandler(image.DisplayProgress(buff)))

	fmt.Println(err)
	// Every single message from Docker starts with { and contains ",
	// so raw JSON will always have {", while formatted output strips
	// away the JSON structure.
	fmt.Println(!strings.Contains(buff.String(), "{\""))

	// Output:
	// <nil>
	// true
}
