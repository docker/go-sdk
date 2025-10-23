package client_test

import (
	"context"
	"fmt"
	"log"

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
