package container_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/go-sdk/container"
	"github.com/docker/go-sdk/container/exec"
)

func ExampleRun() {
	ctr, err := container.Run(context.Background(), container.WithImage("alpine:latest"))
	fmt.Println(err)
	fmt.Println(ctr.ID() != "")

	err = ctr.Terminate(context.Background())
	fmt.Println(err)

	// Output:
	// <nil>
	// true
	// <nil>
}

func ExampleContainer_Terminate() {
	ctr, err := container.Run(context.Background(), container.WithImage("alpine:latest"))
	fmt.Println(err)

	err = ctr.Terminate(context.Background())
	fmt.Println(err)

	// Output:
	// <nil>
	// <nil>
}

func ExampleContainer_lifecycle() {
	ctr, err := container.Run(
		context.Background(),
		container.WithImage("alpine:latest"),
		container.WithNoStart(),
	)

	fmt.Println(err)

	err = ctr.Start(context.Background())
	fmt.Println(err)

	err = ctr.Stop(context.Background())
	fmt.Println(err)

	err = ctr.Start(context.Background())
	fmt.Println(err)

	err = ctr.Terminate(context.Background())
	fmt.Println(err)

	// Output:
	// <nil>
	// <nil>
	// <nil>
	// <nil>
	// <nil>
}

func ExampleContainer_Inspect() {
	ctr, err := container.Run(context.Background(), container.WithImage("alpine:latest"))
	fmt.Println(err)

	inspect, err := ctr.Inspect(context.Background())
	fmt.Println(err)
	fmt.Println(inspect.ID != "")

	err = ctr.Terminate(context.Background())
	fmt.Println(err)

	// Output:
	// <nil>
	// <nil>
	// true
	// <nil>
}

func ExampleContainer_Logs() {
	ctr, err := container.Run(context.Background(), container.WithImage("hello-world:latest"))
	fmt.Println(err)

	logs, err := ctr.Logs(context.Background())
	fmt.Println(err)

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, logs)
	fmt.Println(err)
	fmt.Println(strings.Contains(buf.String(), "Hello from Docker!"))

	err = ctr.Terminate(context.Background())
	fmt.Println(err)

	// Output:
	// <nil>
	// <nil>
	// <nil>
	// true
	// <nil>
}

func ExampleContainer_copy() {
	ctr, err := container.Run(context.Background(), container.WithImage("alpine:latest"))
	fmt.Println(err)

	content := []byte("Hello, World!")

	err = ctr.CopyToContainer(context.Background(), content, "/tmp/test.txt", 0o644)
	fmt.Println(err)

	rc, err := ctr.CopyFromContainer(context.Background(), "/tmp/test.txt")
	fmt.Println(err)

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, rc)
	fmt.Println(err)
	fmt.Println(buf.String())

	err = ctr.Terminate(context.Background())
	fmt.Println(err)

	// Output:
	// <nil>
	// <nil>
	// <nil>
	// <nil>
	// Hello, World!
	// <nil>
}

func ExampleContainer_Exec() {
	ctr, err := container.Run(context.Background(), container.WithImage("nginx:alpine"))
	fmt.Println(err)

	code, rc, err := ctr.Exec(
		context.Background(),
		[]string{"pwd"},
		exec.Multiplexed(),
		exec.WithWorkingDir("/usr/share/nginx/html"),
	)
	fmt.Println(err)
	fmt.Println(code)

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, rc)
	fmt.Println(err)
	fmt.Print(buf.String()) // not adding a newline to the output

	err = ctr.Terminate(context.Background())
	fmt.Println(err)

	// Output:
	// <nil>
	// <nil>
	// 0
	// <nil>
	// /usr/share/nginx/html
	// <nil>
}

func ExampleFromResponse() {
	// First, create a container using Run
	ctr, err := container.Run(context.Background(), container.WithImage("alpine:latest"))
	if err != nil {
		fmt.Println(err)
		return
	}

	// Use the SDK client from the existing container
	cli := ctr.Client()

	// List containers to get the Summary (this is what you'd typically get from the Docker API)
	containers, err := cli.ContainerList(context.Background(), containertypes.ListOptions{All: true})
	if err != nil {
		fmt.Println(err)
		return
	}

	// Find our container in the list
	var summary containertypes.Summary
	for _, c := range containers {
		if c.ID == ctr.ID() {
			summary = c
			break
		}
	}

	// Now recreate the container using FromResponse with the container summary
	// This is useful when you only have a container ID and need to perform operations on it
	recreated, err := container.FromResponse(context.Background(), cli, summary)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Container IDs match:", recreated.ID() == ctr.ID())

	// Now you can use operations like CopyToContainer on the recreated container
	content := []byte("Hello from FromResponse!")
	if err := recreated.CopyToContainer(context.Background(), content, "/tmp/test.txt", 0o644); err != nil {
		fmt.Println(err)
		return
	}

	// Verify the file was copied
	rc, err := recreated.CopyFromContainer(context.Background(), "/tmp/test.txt")
	if err != nil {
		fmt.Println(err)
		return
	}
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, rc); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("File content:", buf.String())

	// Terminate the recreated container
	err = recreated.Terminate(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}

	// Terminate the original container should fail
	err = ctr.Terminate(context.Background())
	if err == nil {
		// Termination unexpectedly succeeded; a failure path for the example
		return
	}
	fmt.Println("Container did not exist")

	// Output:
	// Container IDs match: true
	// File content: Hello from FromResponse!
	// Container did not exist
}
