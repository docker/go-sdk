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

func ExampleSDKClient_FindContainerByID() {
	ctx := context.Background()

	cli, err := client.New(ctx)
	if err != nil {
		fmt.Println("skip")
		return
	}

	if _, err := cli.Ping(ctx, dockerclient.PingOptions{}); err != nil {
		fmt.Println("skip")
		return
	}

	res, err := cli.ContainerCreate(ctx, dockerclient.ContainerCreateOptions{
		Config: &container.Config{
			Image: "nginx:alpine",
		},
	})
	if err != nil {
		fmt.Println("skip")
		return
	}
	defer func() {
		_, err := cli.ContainerRemove(ctx, res.ID, dockerclient.ContainerRemoveOptions{
			Force: true,
		})
		if err != nil {
			fmt.Println("error removing container")
		}
	}()

	c, err := cli.FindContainerByID(ctx, res.ID)
	if err != nil {
		fmt.Println("error finding container by ID")
		return
	}

	fmt.Println(c.ID == res.ID)

	// Output:
	// true
}
