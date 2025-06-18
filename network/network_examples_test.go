package network_test

import (
	"context"
	"fmt"
	"runtime"

	apinetwork "github.com/docker/docker/api/types/network"
	"github.com/docker/go-sdk/client"
	"github.com/docker/go-sdk/network"
)

func ExampleNew() {
	nw, err := network.New(context.Background())
	fmt.Println(err)
	fmt.Println(nw.Name() != "")

	err = nw.Terminate(context.Background())
	fmt.Println(err)

	// Output:
	// <nil>
	// true
	// <nil>
}

func ExampleNew_withClient() {
	dockerClient, err := client.New(context.Background())
	fmt.Println(err)

	nw, err := network.New(context.Background(), network.WithClient(dockerClient))
	fmt.Println(err)
	fmt.Println(nw.Name() != "")

	err = nw.Terminate(context.Background())
	fmt.Println(err)

	// Output:
	// <nil>
	// <nil>
	// true
	// <nil>
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
	fmt.Println(err)

	fmt.Println(nw.Name())
	fmt.Println(nw.Driver() != "")

	err = nw.Terminate(context.Background())
	fmt.Println(err)

	// Output:
	// <nil>
	// test-network
	// true
	// <nil>
}

func ExampleNetwork_Inspect() {
	name := "test-network-inspect"
	nw, err := network.New(context.Background(), network.WithName(name))
	fmt.Println(err)

	inspect, err := nw.Inspect(context.Background())
	fmt.Println(err)
	fmt.Println(inspect.Name)

	err = nw.Terminate(context.Background())
	fmt.Println(err)

	// Output:
	// <nil>
	// <nil>
	// test-network-inspect
	// <nil>
}

func ExampleNetwork_Inspect_withOptions() {
	name := "test-network-inspect-options"
	nw, err := network.New(context.Background(), network.WithName(name))
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

	err = nw.Terminate(context.Background())
	fmt.Println(err)

	// Output:
	// <nil>
	// <nil>
	// test-network-inspect-options
	// <nil>
}

func ExampleNetwork_Terminate() {
	nw, err := network.New(context.Background())
	fmt.Println(err)

	err = nw.Terminate(context.Background())
	fmt.Println(err)

	// Output:
	// <nil>
	// <nil>
}
