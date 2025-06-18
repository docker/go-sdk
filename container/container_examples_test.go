package container_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

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
