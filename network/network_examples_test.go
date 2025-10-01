package network_test

import (
	"context"
	"fmt"
	"runtime"

	"github.com/docker/docker/api/types/filters"
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

func ExampleNetwork_Client_networksPrune() {
	nw, err := network.New(context.Background(), network.WithName("test-network-create"))
	if err != nil {
		fmt.Println(err)
		return
	}

	inspect, err := network.FindByID(context.Background(), nw.ID())
	if err != nil {
		fmt.Println(err)
		err = nw.Terminate(context.Background())
		if err != nil {
			fmt.Println(err)
		}
		return
	}

	fmt.Println(inspect.Name)

	sdkClient := nw.Client()

	f := filters.NewArgs()
	for k, v := range client.SDKLabels() {
		f.Add("label", k+"="+v)
	}

	report, err := sdkClient.NetworksPrune(context.Background(), f)
	if err != nil {
		fmt.Println(err)
		err = nw.Terminate(context.Background())
		if err != nil {
			fmt.Println(err)
		}
		return
	}
	fmt.Println(report.NetworksDeleted)

	// Output:
	// test-network-create
	// [test-network-create]
}
