# ⚠️ Disclaimer

**This repository is Work in Progress (WIP)**. We are actively developing and improving this SDK. While the current functionality is stable and continuously tested in the CI, the API may change as we continue to enhance and refine the codebase until we reach a `v1.0.0` release. We recommend:

- Using specific version tags in your dependencies.
- Reviewing the changelog before upgrading.
- Testing thoroughly in your environment.
- Reporting any issues you encounter.

# Docker SDK for Go

A lightweight, modular SDK for interacting with Docker configuration and context data in Go.

This project is designed to be:
- **Extensible**: Built with composability in mind to support additional Docker-related features.
- **Lightweight**: No unnecessary dependencies; only what's needed to manage Docker configurations.
- **Go-native**: Idiomatic Go modules for clean integration in CLI tools and backend services.

### Why should I use this SDK instead of the moby/moby/client package?

The [`moby/moby/client`](https://github.com/moby/moby/tree/master/client) package is a low-level client that provides a direct interface to the Docker daemon. On the other hand, the `go-sdk` is a higher-level client that provides a more convenient interface to the Docker daemon, simplifying the interactions in operations like pulling images with authentication or running containers. It's simpler because it aggregates the most used operations into a single API call, which results in less code to read, write and maintain.

At the same time, the `go-sdk` exposes the low-level Moby client, allowing you to use it in more complex scenarios when you need to interact with the Docker daemon in a more advanced low-level way.

Also, the `go-sdk` provides the following features that enhances the experience of running containers:

- **Random ports** for exposed ports, so you don't need to worry about port conflicts
- **Wait strategies**, so you don't need to worry about waiting for the container to be ready. The readiness can be defined by using the existing wait strategies, such as for a given port to be listening, a given log happening in the container logs, an HTTP request, a command to exit, etc.

Here's a table that summarizes the differences between the two Docker clients:

| Feature/Characteristic | `moby/moby/client` | `docker/go-sdk` |
|------------------------|-------------------|----------|
| **Interface Level** | Low-level | High-level |
| **Direct Docker Daemon Access** | ✅ | ✅ (by exposing the low-level `moby/moby/client`) |
| **Simplified Operations by Aggregating API Calls** | | ✅ |
| **Less Code to Maintain** | | ✅ |
| **Complex Scenario Support** | ✅ | ✅ |
| **Discovers Authentication for Pulling Images** | | ✅ |
| **Functional Options** | | ✅ |
| **Random Port Assignment** | | ✅ |
| **Wait Strategies** | | ✅ |

The `go-sdk` project contains lots of testable examples, so feel free to use it as a reference for comparing with your current usage of the `moby/moby/client` package. To name a few:

#### Running a container

With the `moby/moby/client` package, you would need to:

<details>
  <summary>See the code</summary>

```go
package main

import (
	"context"
	"io"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

func main() {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	reader, err := cli.ImagePull(ctx, "docker.io/library/alpine", image.PullOptions{})
	if err != nil {
		panic(err)
	}

	defer reader.Close()
	// cli.ImagePull is asynchronous.
	// The reader needs to be read completely for the pull operation to complete.
	// If stdout is not required, consider using io.Discard instead of os.Stdout.
	io.Copy(os.Stdout, reader)

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "alpine",
		Cmd:   []string{"echo", "hello world"},
		Tty:   false,
	}, nil, nil, nil, "")
	if err != nil {
		panic(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		panic(err)
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			panic(err)
		}
	case <-statusCh:
	}

	out, err := cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true})
	if err != nil {
		panic(err)
	}

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)
}
```

</details>

With the `go-sdk`, you can do:

<details>
  <summary>See the code</summary>

```go
package main

import (
	"context"
	"os"

	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-sdk/container"
	"github.com/docker/go-sdk/container/wait"
)

func main() {
	ctr, err := container.Run(
		context.Background(),
		container.WithImage("alpine:latest"),
		container.WithCmd("echo", "hello world"),
		container.WithWaitStrategy(wait.ForLog("hello world")),
	)
	if err != nil {
		panic(err)
	}

	logs, err := ctr.Logs(context.Background())
	if err != nil {
		panic(err)
	}

	stdcopy.StdCopy(os.Stdout, os.Stderr, logs)

	err = ctr.Terminate(context.Background())
	if err != nil {
		panic(err)
	}
}
```

</details>

#### Pulling an image

With the `moby/moby/client` package, you would need to:

<details>
  <summary>See the code</summary>

```go
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
)

func main() {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	authConfig := registry.AuthConfig{
		Username: "username",
		Password: "password",
	}
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		panic(err)
	}
	authStr := base64.URLEncoding.EncodeToString(encodedJSON)

	out, err := cli.ImagePull(ctx, "my-registry.com/alpine", image.PullOptions{RegistryAuth: authStr})
	if err != nil {
		panic(err)
	}

	defer out.Close()
	io.Copy(os.Stdout, out)
}
```

</details>

With the `go-sdk`, as soon as the current Docker config has an entry for the private registry, you can do:

<details>
  <summary>See the code</summary>

```go
package main

import (
	"context"
	"fmt"

	"github.com/docker/go-sdk/image"
)

func main() {
	err := image.Pull(context.Background(), "my-registry.com/alpine")
	if err != nil {
		panic(err)
	}
}
```

</details>

#### Reading the current Docker context

With the `moby/moby/client` package, you basically can't do it, as this functionality is part of the client code of the Docker CLI.

With the `go-sdk`, you can do:

<details>
  <summary>See the code</summary>

```go
package main

import (
	"fmt"

	"github.com/docker/go-sdk/context"
)

func main() {
	ctx, err := context.Current()
	if err != nil {
		panic(err)
	}
	fmt.Println("Current Docker context name:", ctx)
}
```

</details>

## Features

- Initialize a Docker client, using the current Docker context to resolve the Docker host and socket
- Parse and load Docker CLI config (`~/.docker/config.json`)
- Handle credential helpers
- Read and manage Docker contexts
- Pull images from a remote registry, retrying on non-permanent errors

## Installation

```bash
go get github.com/docker/go-sdk/client
go get github.com/docker/go-sdk/config
go get github.com/docker/go-sdk/container
go get github.com/docker/go-sdk/context
go get github.com/docker/go-sdk/image
go get github.com/docker/go-sdk/network
go get github.com/docker/go-sdk/volume
```

## Usage

### client

Using the default client:

```go
cli := client.DefaultClient
```

Creating a new client, with optional configuration:

```go
cli, err := client.New(context.Background(), client.WithDockerContext("my-docker-context"))
if err != nil {
	log.Fatalf("failed to create docker client: %v", err)
}

// Close the docker client when done
defer cli.Close()
```

Please refer to the [client](./client/README.md) package for more information.

### config

```go
cfg, err := config.Load()
if err != nil {
	log.Fatalf("failed to load config: %v", err)
}

auth, ok := cfg.AuthConfigs["https://index.docker.io/v1/"]
if ok {
	fmt.Println("Username:", auth.Username)
}
```

### container

```go
ctr, err := container.Run(context.Background(),
	container.WithImage("nginx:alpine"),
	container.WithImagePlatform("linux/amd64"),
	container.WithAlwaysPull(),
	container.WithExposedPorts("80/tcp"),
	container.WithWaitStrategy(wait.ForListeningPort("80/tcp")),
)
if err != nil {
	log.Fatalf("failed to run container: %v", err)
}

container.Terminate(ctr)
```

### context

```go
current, err := context.Current()
if err != nil {
	log.Fatalf("failed to get current docker context: %v", err)
}

fmt.Printf("current docker context: %s", current)

dockerHost, err := context.CurrentDockerHost()
if err != nil {
	log.Fatalf("failed to get current docker host: %v", err)
}
fmt.Printf("current docker host: %s", dockerHost)
```

### image

```go
import (
	"context"

	apiimage "github.com/docker/docker/api/types/image"
	"github.com/docker/go-sdk/client"
	"github.com/docker/go-sdk/image"
)

ctx := context.Background()
dockerClient, err := client.New(ctx)
if err != nil {
	log.Fatalf("failed to create docker client: %v", err)
}
defer dockerClient.Close()

err = image.Pull(ctx,
	"nginx:alpine",
	image.WithPullClient(dockerClient),
	image.WithPullOptions(apiimage.PullOptions{}),
)
if err != nil {
	log.Fatalf("failed to pull image: %v", err)
}

```

### network

```go
nw, err := network.New(ctx)
if err != nil {
	log.Fatalf("failed to create network: %v", err)
}

resp, err := nw.Inspect(ctx)
if err != nil {
	log.Fatalf("failed to inspect network: %v", err)
}

err = nw.Terminate(ctx)
if err != nil {
	log.Fatalf("failed to terminate network: %v", err)
}
```

### volume

```go
v, err := volume.New(ctx)
if err != nil {
	log.Fatalf("failed to create volume: %v", err)
}

vol, err := volume.FindByID(ctx, v.ID())
if err != nil {
	log.Println(err)
	return
}

err = v.Terminate(ctx)
if err != nil {
	log.Fatalf("failed to terminate volume: %v", err)
}
```

More usage examples are coming soon!

## Contributing

We welcome contributions! Please read the [CONTRIBUTING](./CONTRIBUTING.md) file and open issues or submit pull requests once you're ready. Make sure your changes are well-tested and documented.

## Licensing

This project is licensed under the [Apache License 2.0](./LICENSE).

It includes portions of code derived from other open source projects which are licensed under the MIT License. Their original licenses are preserved [here](./third_party), and attribution is provided in the [NOTICE](./NOTICE) file.

Modifications have been made to this code as part of its integration into this project.
