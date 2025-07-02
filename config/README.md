# Docker Config

This package provides a simple API to load docker CLI configs, auths, etc. with minimal deps.

This library is a fork of [github.com/cpuguy83/dockercfg](https://github.com/cpuguy83/dockercfg). Read the [NOTICE](../NOTICE) file for more details.

## Installation

```bash
go get github.com/docker/go-sdk/config
```

## Usage

### Docker Config

#### Directory

It will return the current Docker config directory.

```go
dir, err := config.Dir()
if err != nil {
    log.Fatalf("failed to get current docker config directory: %v", err)
}

fmt.Printf("current docker config directory: %s", dir)
```

#### Filepath

It will return the path to the Docker config file.

```go
filepath, err := config.Filepath()
if err != nil {
    log.Fatalf("failed to get current docker config file path: %v", err)
}

fmt.Printf("current docker config file path: %s", filepath)
```

#### Load

It will return the Docker config.

```go
cfg, err := config.Load()
if err != nil {
    log.Fatalf("failed to load docker config: %v", err)
}

fmt.Printf("docker config: %+v", cfg)
```

### Auth

#### AuthConfigs

It will return a maps of the registry credentials for the given Docker images, indexed by the registry hostname.

```go
authConfigs, err := config.AuthConfigs("nginx:latest")
if err != nil {
    log.Fatalf("failed to get registry credentials: %v", err)
}

fmt.Printf("registry credentials: %+v", authConfigs)
```

#### Auth Configs For Hostname

It will return the registry credentials for the given Docker registry.

```go
authConfig, err := config.AuthConfigForHostname("https://index.docker.io/v1/")
if err != nil {
    log.Fatalf("failed to get registry credentials: %v", err)
}

fmt.Printf("registry credentials: %+v", authConfig)
```
