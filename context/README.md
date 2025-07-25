# Docker Contexts

This package provides a simple API to interact with Docker contexts.

## Installation

```bash
go get github.com/docker/go-sdk/context
```

## Usage

### Current Context

It returns the current Docker context name.

```go
current, err := context.Current()
if err != nil {
    log.Fatalf("failed to get current docker context: %v", err)
}

fmt.Printf("current docker context: %s", current)
```

### Current Docker Host

It returns the Docker host that the current context is configured to use.

```go
dockerHost, err := context.CurrentDockerHost()
if err != nil {
    log.Fatalf("failed to get current docker host: %v", err)
}
fmt.Printf("current docker host: %s", dockerHost)
```

### Docker Host From Context

It returns the Docker host that the given context is configured to use.

```go
dockerHost, err := context.DockerHostFromContext("desktop-linux")
if err != nil {
    log.Printf("error getting docker host from context: %s", err)
    return
}

fmt.Printf("docker host from context: %s", dockerHost)
```

### Inspect Context

It returns the description of the given context.

```go
description, err := context.Inspect("context1")
if err != nil {
    log.Printf("failed to inspect context: %v", err)
    return
}

fmt.Printf("description: %s", description)
```

If the context is not found, it returns an `ErrDockerContextNotFound` error.

### List Contexts

It returns the list of contexts available in the Docker configuration.

```go
contexts, err := context.List()
if err != nil {
    log.Printf("failed to list contexts: %v", err)
    return
}

fmt.Printf("contexts: %v", contexts)
```

### Add Context

It adds a new context to the Docker configuration, identified by a name. It's possible to pass options to customize the context definition.

```go
ctx, err := context.New("my-context")
if err != nil {
    log.Printf("failed to add context: %v", err)
    return
}

fmt.Printf("context added: %s", ctx.Name)
```

### Available Options

The following options are available to customize the context definition:

- `WithHost(host string) CreateContextOption` sets the host for the context.
- `WithDescription(description string) CreateContextOption` sets the description for the context.
- `WithAdditionalFields(fields map[string]any) CreateContextOption` sets the additional fields for the context.
- `WithSkipTLSVerify() CreateContextOption` sets the skipTLSVerify flag to true.
- `AsCurrent() CreateContextOption` sets the context as the current context, saving the current context to the Docker configuration.

### Delete Context

It deletes a context from the Docker configuration.

```go
ctx, err := context.New("my-context")
if err != nil {
    log.Printf("error adding context: %s", err)
    return
}

if err := ctx.Delete(); err != nil {
    log.Printf("failed to delete context: %v", err)
    return
}

fmt.Printf("context deleted: %s", ctx.Name)
```