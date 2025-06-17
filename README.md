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

## Features

- Initialize a Docker client, using the current Docker context to resolve the Docker host and socket
- Parse and load Docker CLI config (`~/.docker/config.json`)
- Handle credential helpers
- Read and manage Docker contexts
- Pull images from a remote registry, retrying on non-permanent errors

## Installation

```bash
go get github.com/docker/go-sdk
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

fmt.Printf("network: %+v", resp)

err = nw.Terminate(ctx)
if err != nil {
    log.Fatalf("failed to terminate network: %v", err)
}
```

More usage examples are coming soon!

## Contributing

We welcome contributions! Please read the [CONTRIBUTING](./CONTRIBUTING.md) file and open issues or submit pull requests once you're ready. Make sure your changes are well-tested and documented.

## Licensing

This project is licensed under the [Apache License 2.0](./LICENSE).

It includes portions of code derived from other open source projects which are licensed under the MIT License. Their original licenses are preserved [here](./third_party), and attribution is provided in the [NOTICE](./NOTICE) file.

Modifications have been made to this code as part of its integration into this project.
