package image_test

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"log/slog"
	"path"

	"github.com/docker/docker/api/types/build"
	dockerimage "github.com/docker/docker/api/types/image"
	"github.com/docker/go-sdk/client"
	"github.com/docker/go-sdk/image"
)

func ExampleBuild() {
	// using a buffer to capture the build output
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, nil))

	cli, err := client.New(context.Background(), client.WithLogger(logger))
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

	tag, err := image.Build(
		context.Background(), contextArchive, "example:test",
		image.WithBuildOptions(build.ImageBuildOptions{
			Dockerfile: "Dockerfile",
		}),
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
	// using a buffer to capture the build output
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, nil))

	cli, err := client.New(context.Background(), client.WithLogger(logger))
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

	tag, err := image.BuildFromDir(
		context.Background(), buildPath, "Dockerfile", "example:test",
		image.WithBuildOptions(build.ImageBuildOptions{
			Dockerfile: "Dockerfile",
		}),
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
