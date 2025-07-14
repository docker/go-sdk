package image_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"strings"

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

func ExamplePull_withPullHandler() {
	opts := apiimage.PullOptions{
		Platform: "linux/amd64",
	}

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
