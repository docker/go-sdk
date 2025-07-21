# Legacy Adapters

> **⚠️ DEPRECATION NOTICE**
> 
> This module is **temporary** and already **deprecated**. It will be removed in a future release when all Docker products have migrated to use the go-sdk natively.
> 
> **We strongly recommend avoiding this module in new projects.** Instead, use the native go-sdk types directly.   
> This module exists solely to provide a migration path for existing Docker products during the transition period.

This package provides conversion utilities to bridge between the modern Docker Go SDK types and legacy Docker CLI/Docker Engine API types.

## Installation

```bash
go get github.com/docker/go-sdk/legacyadapters
```

## Usage

### Converting Auth Configuration

Convert SDK auth config to Docker Engine API format:

```go
import (
    "github.com/docker/go-sdk/config"
    legacyconfig "github.com/docker/go-sdk/legacyadapters/config"
)

sdkAuth := config.AuthConfig{
    Username:      "myuser",
    Password:      "mypass",
    ServerAddress: "registry.example.com",
}

// Convert to Docker Engine API format
registryAuth := legacyconfig.ToRegistryAuthConfig(sdkAuth)

// Convert to Docker CLI format
cliAuth := legacyconfig.ToCLIAuthConfig(sdkAuth)
```

### Converting Full Configuration

Convert SDK config to Docker CLI config file format:

```go
sdkConfig := config.Config{
    AuthConfigs: map[string]config.AuthConfig{
        "registry.example.com": {
            Username: "user",
            Password: "pass",
        },
    },
    HTTPHeaders: map[string]string{
        "User-Agent": "my-app/1.0",
    },
    CredentialsStore: "desktop",
}

// Convert to Docker CLI config file format
cliConfigFile := legacyconfig.ToConfigFile(sdkConfig)
```

### Converting Proxy Configuration

```go
sdkProxy := config.ProxyConfig{
    HTTPProxy:  "http://proxy.example.com:8080",
    HTTPSProxy: "https://proxy.example.com:8443",
    NoProxy:    "localhost,127.0.0.1",
}

// Convert to CLI proxy format
cliProxy := legacyconfig.ToCLIProxyConfig(sdkProxy)

// Convert multiple proxy configs
sdkProxies := map[string]config.ProxyConfig{
    "default": sdkProxy,
}
cliProxies := legacyconfig.ToCLIProxyConfigs(sdkProxies)
```

## Available Functions

### Auth Configuration Adapters

- `ToRegistryAuthConfig`: Converts SDK `AuthConfig` to Docker Engine API `registry.AuthConfig`
- `ToCLIAuthConfig`: Converts SDK `AuthConfig` to Docker CLI `types.AuthConfig`
- `ToCLIAuthConfigs`: Converts map of SDK auth configs to map of CLI auth configs

### Configuration File Adapters

- `ToConfigFile`: Converts SDK `Config` to Docker CLI `configfile.ConfigFile`

### Proxy Configuration Adapters

- `ToCLIProxyConfig`: Converts SDK `ProxyConfig` to CLI `configfile.ProxyConfig`
- `ToCLIProxyConfigs`: Converts map of SDK proxy configs to map of CLI proxy configs

## Key Differences

- **Email Field**: Always set to empty string in converted formats