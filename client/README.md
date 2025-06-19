# Docker Client

This package provides a client for the Docker API.

## Installation

```bash
go get github.com/docker/go-sdk/client
```

## Usage

The library provides a default client that is initialised with the current docker context. It uses a default logger that is configured to print to the standard output using the `slog` package.

```go
cli := client.DefaultClient
```

It's also possible to create a new client, with optional configuration:

```go
cli, err := client.New(context.Background())
if err != nil {
    log.Fatalf("failed to create docker client: %v", err)
}

// Close the docker client when done
defer cli.Close()
```

## Customizing the client

The client created with the `New` function can be customized using functional options. The following options are available:

- `WithHealthCheck(healthCheck func(ctx context.Context) func(c *Client) error) ClientOption`: A healthcheck function that is called to check the health of the client. By default, the client uses `Ping` to check the health of the client.
- `WithDockerHost(dockerHost string) ClientOption`: The docker host to use. By default, the client uses the current docker host.
- `WithDockerContext(dockerContext string) ClientOption`: The docker context to use. By default, the client uses the current docker context.

In the case that both the docker host and the docker context are provided, the docker context takes precedence.
