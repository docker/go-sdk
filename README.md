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

### dockerclient

```go
cli, err := dockerclient.New(context.Background())
if err != nil {
    log.Fatalf("failed to create docker client: %v", err)
}

// Close the docker client when done
defer cli.Close()
```

### dockerconfig

```go
cfg, err := dockerconfig.Load()
if err != nil {
    log.Fatalf("failed to load config: %v", err)
}

auth, ok := cfg.AuthConfigs["https://index.docker.io/v1/"]
if ok {
    fmt.Println("Username:", auth.Username)
}
```

### dockercontainer

```go
ctr, err := dockercontainer.Run(context.Background(),
    dockercontainer.WithImage("nginx:alpine"),
    dockercontainer.WithImagePlatform("linux/amd64"),
    dockercontainer.WithAlwaysPull(),
    dockercontainer.WithExposedPorts("80/tcp"),
    dockercontainer.WithWaitStrategy(wait.ForListeningPort("80/tcp")),
)
if err != nil {
    log.Fatalf("failed to run container: %v", err)
}

dockercontainer.TerminateContainer(ctr)
```

### dockercontext

```go
dockerHost, err := dockercontext.CurrentDockerHost()
if err != nil {
    log.Fatalf("failed to get current docker host: %v", err)
}
```

### dockerimage

```go
err := dockerimage.Pull(ctx, mockImageClient, "someTag", image.PullOptions{})
if err != nil {
    log.Fatalf("failed to pull image: %v", err)
}

```

### dockernetwork

```go
nw, err := dockernetwork.New(ctx)
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
