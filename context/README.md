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