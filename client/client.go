package client

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
)

var (
	defaultLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

	defaultUserAgent = "docker-go-sdk/" + Version()

	defaultHealthCheck = func(ctx context.Context) func(c SDKClient) error {
		return func(c SDKClient) error {
			var pingErr error
			for i := range 3 {
				if _, pingErr = c.Ping(ctx); pingErr == nil {
					return nil
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(time.Millisecond * time.Duration(i+1) * 100):
				}
			}
			return fmt.Errorf("docker daemon not ready: %w", pingErr)
		}
	}
)

// New returns a new client for interacting with containers.
// The client is configured using the provided options, that must be compatible with
// docker's [client.Opt] type.
//
// The Docker host is automatically resolved reading it from the current docker context;
// in case you need to pass [client.Opt] options that override the docker host, you can
// do so by providing the [FromDockerOpt] options adapter.
// E.g.
//
//	cli, err := client.New(context.Background(), client.FromDockerOpt(client.WithHost("tcp://foobar:2375")))
//
// The client uses a logger that is initialized to [io.Discard]; you can change it by
// providing the [WithLogger] option.
// E.g.
//
//	cli, err := client.New(context.Background(), client.WithLogger(slog.Default()))
//
// The client is safe for concurrent use by multiple goroutines.
func New(ctx context.Context, options ...ClientOption) (SDKClient, error) {
	c := &sdkClient{
		log:         defaultLogger,
		healthCheck: defaultHealthCheck,
	}

	cli, err := command.NewDockerCli(command.WithUserAgent(defaultUserAgent))
	if err != nil {
		return nil, err
	}

	err = cli.Initialize(flags.NewClientOptions())
	if err != nil {
		return nil, err
	}
	c.APIClient = cli.Client()
	c.config = cli.ConfigFile()

	for _, opt := range options {
		if err := opt.Apply(c); err != nil {
			return nil, fmt.Errorf("apply option: %w", err)
		}
	}

	if err := c.healthCheck(ctx)(c); err != nil {
		return nil, fmt.Errorf("health check: %w", err)
	}

	return c, nil
}
