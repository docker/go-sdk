package image_test

import (
	"bytes"
	"context"
	"fmt"
	"log"

	"github.com/docker/docker/api/types/build"
	dockerimage "github.com/docker/docker/api/types/image"
	"github.com/docker/go-sdk/client"
	"github.com/docker/go-sdk/image"
)

func ExampleBuild() {
	files := []image.BuildFile{
		{
			Name:    "say_hi.sh",
			Content: []byte(`echo hi this is from the say_hi.sh file!`),
		},
		{
			Name: "Dockerfile",
			Content: []byte(`FROM alpine
					WORKDIR /app
					COPY . .
					CMD ["sh", "./say_hi.sh"]`),
		},
	}

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

	reader, err := image.ReaderFromFiles(files)
	if err != nil {
		log.Println("error creating reader", err)
		return
	}

	// using a buffer to capture the build output
	buf := &bytes.Buffer{}

	tag, err := image.Build(
		context.Background(), reader, "example:test",
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
