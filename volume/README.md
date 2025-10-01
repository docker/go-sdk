# Docker Volumes

This package provides a simple API to create and manage Docker volumes.

## Installation

```bash
go get github.com/docker/go-sdk/volume
```

## Usage

```go
v, err := volume.New(context.Background(), volume.WithName("my-volume-list"), volume.WithLabels(map[string]string{"volume.type": "example-test"}))
if err != nil {
    log.Println(err)
    return
}
defer func() {
    if err := v.Terminate(context.Background()); err != nil {
        log.Println(err)
    }
}()
fmt.Printf("volume: %+v", vol)

vol, err := volume.FindByID(context.Background(), v.ID())
if err != nil {
    log.Println(err)
    return
}
fmt.Printf("volume: %+v", vol)

vols, err := volume.List(context.Background(), volume.WithFilters(filters.NewArgs(filters.Arg("label", "volume.type=example-test"))))
if err != nil {
    log.Println(err)
    return
}

fmt.Println(len(vols))
for _, v := range vols {
    fmt.Printf("%s", v.Name)
}

err = v.Terminate(ctx)
if err != nil {
    log.Fatalf("failed to terminate volume: %v", err)
}
```

## Customizing the volume

The volume created with the `New` function can be customized using functional options. The following options are available:

- `WithClient(client client.SDKClient) volume.Option`: The client to use to create the volume. If not provided, the default client will be used.
- `WithName(name string) volume.Option`: The name of the volume.
- `WithLabels(labels map[string]string) volume.Option`: The labels of the volume.

When terminating a volume, the `Terminate` function can be customized using functional options. The following options are available:

- `WithForce() volume.TerminateOption`: Whether to force the termination of the volume.

When finding a volume, the `FindByID` and `List` functions can be customized using functional options. The following options are available:

- `WithFindClient(client *client.Client) volume.FindOptions`: The client to use to find the volume. If not provided, the default client will be used.
- `WithFilters(filters filters.Args) volume.FindOptions`: The filters to use to find the volume. In the case of the `FindByID` function, this option is ignored.
