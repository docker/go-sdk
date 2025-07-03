package image_test

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"path"

	"github.com/docker/docker/api/types/build"
	dockerimage "github.com/docker/docker/api/types/image"
	"github.com/docker/go-sdk/client"
	"github.com/docker/go-sdk/image"
)

func ExampleBuild() {
	cli, err := client.New(context.Background())
	if err != nil {
		log.Println("error creating docker client", err)
		return
	}
	defer func() {
		err := cli.Close()
		if err != nil {
			log.Println("error closing docker client", err)
		}
	}()

	buildPath := path.Join("testdata", "build")

	contextArchive, err := image.ArchiveBuildContext(buildPath, "Dockerfile")
	if err != nil {
		log.Println("error creating reader", err)
		return
	}

	// using a buffer to capture the build output
	buf := &bytes.Buffer{}

	tag, err := image.Build(
		context.Background(), contextArchive, "example:test",
		image.WithBuildOptions(build.ImageBuildOptions{
			Dockerfile: "Dockerfile",
		}),
		image.WithLogWriter(buf),
	)
	if err != nil {
		log.Println("error building image", err)
		return
	}
	defer func() {
		_, err = image.Remove(context.Background(), tag, image.WithRemoveOptions(dockerimage.RemoveOptions{
			Force:         true,
			PruneChildren: true,
		}))
		if err != nil {
			log.Println("error removing image", err)
		}
	}()

	fmt.Println(tag)

	// Output:
	// example:test
}

func ExampleBuildFromDir() {
	cli, err := client.New(context.Background())
	if err != nil {
		log.Println("error creating docker client", err)
		return
	}
	defer func() {
		err := cli.Close()
		if err != nil {
			log.Println("error closing docker client", err)
		}
	}()

	buildPath := path.Join("testdata", "build")

	// using a buffer to capture the build output
	buf := &bytes.Buffer{}

	tag, err := image.BuildFromDir(
		context.Background(), buildPath, "Dockerfile", "example:test",
		image.WithBuildOptions(build.ImageBuildOptions{
			Dockerfile: "Dockerfile",
		}),
		image.WithLogWriter(buf),
	)
	if err != nil {
		log.Println("error building image", err)
		return
	}
	defer func() {
		_, err = image.Remove(context.Background(), tag, image.WithRemoveOptions(dockerimage.RemoveOptions{
			Force:         true,
			PruneChildren: true,
		}))
		if err != nil {
			log.Println("error removing image", err)
		}
	}()

	fmt.Println(tag)

	// Output:
	// example:test
}
