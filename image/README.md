# Docker Images

This package provides a simple API to create and manage Docker images.

## Installation

```bash
go get github.com/docker/go-sdk/image
```

## Usage

```go
err = image.Pull(ctx, "nginx:alpine")
if err != nil {
    log.Fatalf("failed to pull image: %v", err)
}
```

## Customizing the Pull operation

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
