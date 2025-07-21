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
- `WithPullHandler(pullHandler func(r io.ReadCloser) error) image.PullOption`: The handler to use to pull the image, which acts as a callback to the pull operation.

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
    image.WithPullHandler(func(r io.ReadCloser) error {
        // do something with the reader
        return nil
    }),
)
if err != nil {
    log.Fatalf("failed to pull image: %v", err)
}
```

## Removing images

### Usage

```go
err = image.Remove(ctx, "nginx:alpine")
if err != nil {
    log.Fatalf("failed to remove image: %v", err)
}
```

### Customizing the Remove operation

The Remove operation can be customized using functional options. The following options are available:

- `WithRemoveClient(client *client.Client) image.RemoveOption`: The client to use to remove the image. If not provided, the default client will be used.
- `WithRemoveOptions(options dockerimage.RemoveOptions) image.RemoveOption`: The options to use to remove the image. The type of the options is "github.com/docker/docker/api/types/image".

First, you need to import the following packages:

```go
import (
	"context"

    dockerimage "github.com/docker/docker/api/types/image"
	"github.com/docker/go-sdk/client"
	"github.com/docker/go-sdk/image"
)
```

In your code:

```go
ctx := context.Background()
dockerClient, err := client.New(ctx)
if err != nil {
    log.Println("failed to create docker client", err)
    return
}
defer dockerClient.Close()

resp, err := image.Remove(ctx, img, image.WithRemoveOptions(dockerimage.RemoveOptions{
    Force:         true,
    PruneChildren: true,
}))
if err != nil {
    log.Println("failed to remove image", err)
    return
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

err = image.Build(ctx, contextArchive, "example:test",
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

## Extracting images from a Dockerfile

There are three functions to extract images from a Dockerfile:

- `ImagesFromDockerfile(dockerfile string, buildArgs map[string]*string) ([]string, error)`: Extracts images from a Dockerfile.
- `ImagesFromReader(r io.Reader, buildArgs map[string]*string) ([]string, error)`: Extracts images from a Dockerfile reader.
- `ImagesFromTarReader(r io.ReadSeeker, dockerfile string, buildArgs map[string]*string) ([]string, error)`: Extracts images from a Dockerfile reader that is a tar reader.

A Dockerfile can exist in different formats:

- A single Dockerfile file.
- A Dockerfile in a reader, which can be a file or a buffer.
- A Dockerfile inside a tar reader, as part of a build context.

The first two cases are handled by the `ImagesFromDockerfile` and `ImagesFromReader` functions.

```go
images, err := image.ImagesFromDockerfile("Dockerfile", nil)
if err != nil {
    log.Println("error extracting images", err)
    return
}
```

The `ImagesFromTarReader` function is useful when the Dockerfile is inside a tar reader, as part of a build context.

```go
images, err := image.ImagesFromTarReader(contextArchive, "Dockerfile", nil)
if err != nil {
    log.Println("error extracting images", err)
    return
}
```

In this case, the `contextArchive` is a tar reader, and the `Dockerfile` is the path to the Dockerfile inside the tar reader.
