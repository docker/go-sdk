# Docker Images

This package provides a simple API to create and manage Docker images.

## Installation

```bash
go get github.com/docker/go-sdk/image
```

## Pulling images

### Usage

```go
err = image.Pull(ctx, "nginx:alpine")
if err != nil {
    log.Fatalf("failed to pull image: %v", err)
}
```

### Customizing the Pull operation

The Pull operation can be customized using functional options. The following options are available:

- `WithPullClient(client *client.Client) image.PullOption`: The client to use to pull the image. If not provided, the default client will be used.
- `WithPullOptions(options apiimage.PullOptions) image.PullOption`: The options to use to pull the image. The type of the options is "github.com/docker/docker/api/types/image".

First, you need to import the following packages:
```go
import (
	"context"

    apiimage "github.com/docker/docker/api/types/image"
	"github.com/docker/go-sdk/client"
	"github.com/docker/go-sdk/image"
)
```

And in your code:

```go
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

## Building images

### Usage

```go
// path to the build context
buildPath := path.Join("testdata", "build")

// create a reader from the build context
contextArchive, err := image.ArchiveBuildContext(buildPath, "Dockerfile")
if err != nil {
    log.Println("error creating reader", err)
    return
}

// using a buffer to capture the build output
buf := &bytes.Buffer{}

tag, err := image.Build(
    context.Background(), contextArchive, "example:test",
    image.WithBuildOptions(build.ImageBuildOptions{
        Dockerfile: "Dockerfile",
    }),
    image.WithLogWriter(buf),
)
if err != nil {
    log.Println("error building image", err)
    return
}
```

### Archiving the build context

The build context can be archived using the `ArchiveBuildContext` function. This function will return a reader that can be used to build the image.

```go
buildPath := path.Join("testdata", "build")

contextArchive, err := image.ArchiveBuildContext(buildPath, "Dockerfile")
if err != nil {
    log.Println("error creating reader", err)
    return
}

```

This function needs the relative path to the build context and the Dockerfile path inside the build context. The Dockerfile path is relative to the build context.

### Customizing the Build operation

The Build operation can be customized using functional options. The following options are available:

- `WithBuildClient(client *client.Client) image.BuildOption`: The client to use to build the image. If not provided, the default client will be used.
- `WithLogWriter(writer io.Writer) image.BuildOption`: The writer to use to write the build output. If not provided, the build output will be written to the standard output.
- `WithBuildOptions(options build.ImageBuildOptions) image.BuildOption`: The options to use to build the image. The type of the options is "github.com/docker/docker/api/types/build". If set, the tag and context reader will be overridden with the arguments passed to the `Build` function.

First, you need to import the following packages:

```go
import (
	"context"

    "github.com/docker/docker/api/types/build"
	"github.com/docker/go-sdk/client"
	"github.com/docker/go-sdk/image"
)
```

And in your code:

```go
ctx := context.Background()
dockerClient, err := client.New(ctx)
if err != nil {
    log.Fatalf("failed to create docker client: %v", err)
}
defer dockerClient.Close()

// using a buffer to capture the build output
buf := &bytes.Buffer{}

err = image.Build(
    ctx,
    contextArchive,
    "example:test",
    image.WithBuildClient(dockerClient),
    image.WithBuildOptions(build.ImageBuildOptions{
        Dockerfile: "Dockerfile",
    }),
    image.WithLogWriter(buf),
)
if err != nil {
    log.Println("error building image", err)
    return
}

```