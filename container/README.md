# Docker Containers

This package provides a simple API to create and manage Docker containers.

This library is a fork of [github.com/testcontainers/testcontainers-go](https://github.com/testcontainers/testcontainers-go). Read the [NOTICE](../NOTICE) file for more details.

## Installation

```bash
go get github.com/docker/go-sdk/container
```

## Usage

The `Run` function is the main function to create and manage containers. It can be used to create a new container, start it, wait for it to be ready, and stop it. It receives a Go context, and a variadic list of options to customize the container definition:

```go
err = container.Run(ctx, container.WithImage("nginx:alpine"))
if err != nil {
    log.Fatalf("failed to run container: %v", err)
}
```

## Container Definition

The container definition is a struct that contains the configuration for the container. It represents the container's configuration before it's started, and it's used to create the container in the desired state. You can customize the container definition using functional options when calling the `Run` function. More on this below.

## Container Lifecycle Hooks

The container lifecycle hooks are a set of functions that are called at different stages of the container's lifecycle.

- PreCreate
- PostCreate
- PreStart
- PostStart
- PostReady
- PreStop
- PostStop
- PreTerminate
- PostTerminate

They allow you to customize the container's behavior at different stages of its lifecycle, running custom code before or after the container is created, started, ready, stopped or terminated.

## Copy Files

It's possible to copy files to the container, and this can happen in different stages of the container's lifecycle:

- After the container is created but before it's started: using the `WithFiles` option, you can add files to the container.
- After the container is started: using the container's `CopyToContainer` method, you can copy files to the container.

If you need to copy a directory, you can use the `CopyDirToContainer` method, which uses the parent directory of the container path as the target directory.

It's also possible to copy files from the container to the host, using the container's `CopyFromContainer` method.

## Defining the readiness state for the container

In order to wait for the container to be ready, you can use the `WithWaitStrategy` options, that can be used to define a custom wait strategy for the container. The library provides some predefined wait strategies in the `wait` package:

- ForExec: waits for a command to exit
- ForExit: waits for a container to exit
- ForFile: waits for a file to exist
- ForHealth: waits for a container to be healthy
- ForListeningPort: waits for a port to be listening
- ForHTTP: waits for a container to respond to an HTTP request
- ForLog: waits for a container to log a message
- ForSQL: waits for a SQL connection to be established
- ForAll: waits for a combination of strategies

You can also define your own wait strategy by implementing the `wait.Strategy` interface.

Using wait strategies, you don't need to poll the container state, as the wait strategy will block the execution until the condition is met. This is useful to avoid adding `time.Sleep` to your code, making it more reliable, even on slower systems.

## Customizing the Run function

The Run function can be customized using functional options. The following options are available:

### Available Options

The following options are available to customize the container definition:

- `WithAdditionalLifecycleHooks(hooks ...LifecycleHooks) CustomizeDefinitionOption`
- `WithAdditionalWaitStrategy(strategies ...wait.Strategy) CustomizeDefinitionOption`
- `WithAdditionalWaitStrategyAndDeadline(deadline time.Duration, strategies ...wait.Strategy) CustomizeDefinitionOption`
- `WithAfterReadyCommand(execs ...Executable) CustomizeDefinitionOption`
- `WithAlwaysPull() CustomizeDefinitionOption`
- `WithBridgeNetwork() CustomizeDefinitionOption`
- `WithCmd(cmd ...string) CustomizeDefinitionOption`
- `WithCmdArgs(cmdArgs ...string) CustomizeDefinitionOption`
- `WithConfigModifier(modifier func(config *container.Config)) CustomizeDefinitionOption`
- `WithDockerClient(dockerClient *client.Client) CustomizeDefinitionOption`
- `WithEndpointSettingsModifier(modifier func(settings map[string]*apinetwork.EndpointSettings)) CustomizeDefinitionOption`
- `WithEntrypoint(entrypoint ...string) CustomizeDefinitionOption`
- `WithEntrypointArgs(entrypointArgs ...string) CustomizeDefinitionOption`
- `WithEnv(envs map[string]string) CustomizeDefinitionOption`
- `WithExposedPorts(ports ...string) CustomizeDefinitionOption`
- `WithFiles(files ...File) CustomizeDefinitionOption`
- `WithHostConfigModifier(modifier func(hostConfig *container.HostConfig)) CustomizeDefinitionOption`
- `WithImage(image string) CustomizeDefinitionOption`
- `WithImagePlatform(platform string) CustomizeDefinitionOption`
- `WithImageSubstitutors(fn ...ImageSubstitutor) CustomizeDefinitionOption`
- `WithLabels(labels map[string]string) CustomizeDefinitionOption`
- `WithLifecycleHooks(hooks ...LifecycleHooks) CustomizeDefinitionOption`
- `WithName(containerName string) CustomizeDefinitionOption`
- `WithNetwork(aliases []string, nw *network.Network) CustomizeDefinitionOption`
- `WithNetworkName(aliases []string, networkName string) CustomizeDefinitionOption`
- `WithNewNetwork(ctx context.Context, aliases []string, opts ...network.Option) CustomizeDefinitionOption`
- `WithNoStart() CustomizeDefinitionOption`
- `WithStartupCommand(execs ...Executable) CustomizeDefinitionOption`
- `WithWaitStrategy(strategies ...wait.Strategy) CustomizeDefinitionOption`
- `WithWaitStrategyAndDeadline(deadline time.Duration, strategies ...wait.Strategy) CustomizeDefinitionOption`

Please consider that the options using the `WithAdditional` prefix are cumulative, so you can add multiple options to customize the container definition. On the same hand, the options modifying a map are also cumulative, so you can add multiple options to modify the same map.

For slices, the options are not cumulative, so the last option will override the previous ones. The library offers some helper functions to add elements to the slices, like `WithCmdArgs` or `WithEntrypointArgs`, making them cumulative.

## The Container type

The `Container` type is a struct that represents the created container. It provides methods to interact with the container, such as starting, stopping, executing commands, and accessing logs.

### Available Methods

The following methods are available on the `Container` type:

#### Lifecycle Methods

- `Start(ctx context.Context) error` - Starts the container
- `Stop(ctx context.Context, opts ...StopOption) error` - Stops the container
- `Terminate(ctx context.Context, opts ...TerminateOption) error` - Terminates and removes the container

#### Information Methods

- `ID() string` - Returns the container ID
- `ShortID() string` - Returns the short container ID (first 12 characters)
- `Image() string` - Returns the image used by the container
- `Host(ctx context.Context) (string, error)` - Gets the host of the docker daemon
- `Inspect(ctx context.Context) (*container.InspectResponse, error)` - Inspects the container
- `State(ctx context.Context) (*container.State, error)` - Gets the container state

#### Network Methods

- `ContainerIP(ctx context.Context) (string, error)` - Gets the container's IP address
- `ContainerIPs(ctx context.Context) ([]string, error)` - Gets all container IP addresses
- `NetworkAliases(ctx context.Context) (map[string][]string, error)` - Gets network aliases
- `Networks(ctx context.Context) ([]string, error)` - Gets network names

#### Port Methods

- `MappedPort(ctx context.Context, port nat.Port) (nat.Port, error)` - Gets the mapped port for a container's exposed port

#### Execution Methods

- `Exec(ctx context.Context, cmd []string, options ...exec.ProcessOption) (int, io.Reader, error)` - Executes a command in the container

#### File Operations

- `CopyFromContainer(ctx context.Context, containerFilePath string) (io.ReadCloser, error)` - Copies a file from the container
- `CopyToContainer(ctx context.Context, fileContent []byte, containerFilePath string, fileMode int64) error` - Copies a file to the container

#### Logging Methods

- `Logger() *slog.Logger` - Returns the container's logger, which is a `slog.Logger` instance, set at the Docker client level
- `Logs(ctx context.Context) (io.ReadCloser, error)` - Gets container logs
