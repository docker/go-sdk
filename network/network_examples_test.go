package network_test

import (
	"context"
	"fmt"
	"log"
	"runtime"

	apinetwork "github.com/docker/docker/api/types/network"
	"github.com/docker/go-sdk/client"
	"github.com/docker/go-sdk/network"
)

func ExampleNew() {
	nw, err := network.New(context.Background())
	defer nw.Terminate(context.Background())

	fmt.Println(err)
	fmt.Println(nw.Name() != "")

	// Output:
	// <nil>
	// true
}

func ExampleNew_withClient() {
	dockerClient, err := client.New(context.Background())
	if err != nil {
		log.Printf("error creating client: %s", err)
		return
	}

	nw, err := network.New(context.Background(), network.WithClient(dockerClient))
	defer nw.Terminate(context.Background())

	fmt.Println(err)
	fmt.Println(nw.Name() != "")

	// Output:
	// <nil>
	// true
}

func ExampleNew_withOptions() {
	name := "test-network"

	driver := "bridge"
	if runtime.GOOS == "windows" {
		driver = "nat"
	}

	nw, err := network.New(
		context.Background(),
		network.WithName(name),
		network.WithDriver(driver),
		network.WithLabels(map[string]string{"test": "test"}),
		network.WithAttachable(),
	)
	defer nw.Terminate(context.Background())

	fmt.Println(err)
	fmt.Println(nw.Name())
	fmt.Println(nw.Driver() != "")

	// Output:
	// <nil>
	// test-network
	// true
}

func ExampleNetwork_Inspect() {
	name := "test-network-inspect"
	nw, err := network.New(context.Background(), network.WithName(name))
	defer nw.Terminate(context.Background())

	fmt.Println(err)

	inspect, err := nw.Inspect(context.Background())
	fmt.Println(err)
	fmt.Println(inspect.Name)

	// Output:
	// <nil>
	// <nil>
	// test-network-inspect
}

func ExampleNetwork_Inspect_withOptions() {
	name := "test-network-inspect-options"
	nw, err := network.New(context.Background(), network.WithName(name))
	defer nw.Terminate(context.Background())

	fmt.Println(err)

	inspect, err := nw.Inspect(
		context.Background(),
		network.WithNoCache(),
		network.WithInspectOptions(apinetwork.InspectOptions{
			Verbose: true,
			Scope:   "local",
		}),
	)
	fmt.Println(err)
	fmt.Println(inspect.Name)

	// Output:
	// <nil>
	// <nil>
	// test-network-inspect-options
}

func ExampleNetwork_Terminate() {
	nw, err := network.New(context.Background())
	fmt.Println(err)

	nw.Terminate(context.Background())
	fmt.Println(err)

	// Output:
	// <nil>
	// <nil>
}
