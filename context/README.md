# Docker Contexts

This package provides a simple API to discover Docker contexts.

## Installation

```bash
go get github.com/docker/go-sdk/context
```

## Usage

### Current Context

It will return the current Docker context name.

```go
current, err := context.Current()
if err != nil {
    log.Fatalf("failed to get current docker context: %v", err)
}

fmt.Printf("current docker context: %s", current)
```

### Current Docker Host

It will return the Docker host that the current context is configured to use.

```go
dockerHost, err := context.CurrentDockerHost()
if err != nil {
    log.Fatalf("failed to get current docker host: %v", err)
}
fmt.Printf("current docker host: %s", dockerHost)
```

### Docker Host From Context

It will return the Docker host that the given context is configured to use.

```go
dockerHost, err := context.DockerHostFromContext("desktop-linux")
if err != nil {
    log.Printf("error getting docker host from context: %s", err)
    return
}

fmt.Printf("docker host from context: %s", dockerHost)
```
