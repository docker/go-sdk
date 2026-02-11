package client_test

import (
	"context"
	"fmt"
	"log"

	"github.com/moby/moby/api/types/container"
	dockerclient "github.com/moby/moby/client"

	"github.com/docker/go-sdk/client"
)

func ExampleNew() {
	cli, err := client.New(context.Background())
	if err != nil {
		log.Printf("error creating client: %s", err)
		return
	}

	info, err := cli.Info(context.Background(), dockerclient.InfoOptions{})
	if err != nil {
		log.Printf("error getting info: %s", err)
		return
	}

	fmt.Println(info.Info.OperatingSystem != "")

	// Output:
	// true
}

func ExampleSDKClient_FindContainerByName() {
	cli, err := client.New(context.Background())
	if err != nil {
		log.Printf("error creating client: %s", err)
		return
	}
	res, err := cli.ContainerCreate(context.Background(), dockerclient.ContainerCreateOptions{
		Config: &container.Config{
			Image: "nginx:alpine",
		},
		Name: "container-by-name-example",
	})
	if err != nil {
		log.Printf("error creating container: %s", err)
		return
	}
	defer func() {
		if _, err := cli.ContainerRemove(context.Background(), res.ID, dockerclient.ContainerRemoveOptions{Force: true}); err != nil {
			log.Printf("error removing container: %s", err)
		}
	}()

	found, err := cli.FindContainerByName(context.Background(), "container-by-name-example")
	if err != nil {
		log.Printf("error finding container: %s", err)
		return
	}

	fmt.Println(found.Names[0])

	// Output:
	// /container-by-name-example
}
