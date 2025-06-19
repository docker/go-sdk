# Docker Networks

This package provides a simple API to create and manage Docker networks.

## Installation

```bash
go get github.com/docker/go-sdk/network
```

## Usage

```go
nw, err := network.New(ctx)
if err != nil {
    log.Fatalf("failed to create network: %v", err)
}

resp, err := nw.Inspect(ctx)
if err != nil {
    log.Fatalf("failed to inspect network: %v", err)
}

fmt.Printf("network: %+v", resp)

err = nw.Terminate(ctx)
if err != nil {
    log.Fatalf("failed to terminate network: %v", err)
}
```

## Customizing the network

The network created with the `New` function can be customized using functional options. The following options are available:

- `WithClient(client *client.Client) network.Option`: The client to use to create the network. If not provided, the default client will be used.
- `WithName(name string) network.Option`: The name of the network.
- `WithDriver(driver string) network.Option`: The driver of the network.
- `WithInternal() network.Option`: Whether the network is internal.
- `WithEnableIPv6() network.Option`: Whether the network is IPv6 enabled.
- `WithAttachable() network.Option`: Whether the network is attachable.
- `WithLabels(labels map[string]string) network.Option`: The labels of the network.
- `WithIPAM(ipam *network.IPAM) network.Option`: The IPAM configuration of the network.
