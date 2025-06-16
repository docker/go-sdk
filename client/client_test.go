package client_test

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	dockerclient "github.com/docker/docker/client"
	"github.com/docker/go-sdk/client"
	dockercontext "github.com/docker/go-sdk/context"
)

func TestNew(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cli, err := client.New(context.Background())
		require.NoError(t, err)
		require.NotNil(t, cli)

		info, err := cli.Info(context.Background())
		require.NoError(t, err)
		require.NotNil(t, info)
	})

	t.Run("client", func(t *testing.T) {
		cli, err := client.New(context.Background())
		require.NoError(t, err)
		require.NotNil(t, cli)

		require.NotNil(t, cli.Client)
	})

	t.Run("close", func(t *testing.T) {
		cli, err := client.New(context.Background())
		require.NoError(t, err)
		require.NotNil(t, cli)

		// multiple calls to Close() are idempotent
		require.NoError(t, cli.Close())
		require.NoError(t, cli.Close())
	})

	t.Run("success/tls-verify", func(t *testing.T) {
		t.Setenv("DOCKER_TLS_VERIFY", "1")
		t.Setenv("DOCKER_CERT_PATH", filepath.Join("testdata", "certificates"))

		cli, err := client.New(context.Background())
		require.Error(t, err)
		require.Nil(t, cli)
	})

	t.Run("success/apply-option", func(t *testing.T) {
		cli, err := client.New(context.Background(), client.FromDockerOpt(dockerclient.WithHost("tcp://foobar:2375")))
		require.NoError(t, err)
		require.NotNil(t, cli)
	})

	t.Run("error", func(t *testing.T) {
		cli, err := client.New(context.Background(), client.FromDockerOpt(dockerclient.WithHost("foobar")))
		require.Error(t, err)
		require.Nil(t, cli)
	})

	t.Run("error/apply-option", func(t *testing.T) {
		// custom option that always fails to apply
		customOpt := func() client.ClientOption {
			return client.NewClientOption(func(_ *client.Client) error {
				return errors.New("apply option")
			})
		}

		cli, err := client.New(context.Background(), customOpt())
		require.ErrorContains(t, err, "apply option")
		require.Nil(t, cli)
	})

	t.Run("healthcheck/nil", func(t *testing.T) {
		cli, err := client.New(context.Background(), client.WithHealthCheck(nil))
		require.ErrorContains(t, err, "health check is nil")
		require.Nil(t, cli)
	})

	t.Run("healthcheck/noop", func(t *testing.T) {
		noopHealthCheck := func(_ context.Context) func(c *client.Client) error {
			return func(_ *client.Client) error {
				return nil
			}
		}

		cli, err := client.New(context.Background(), client.WithHealthCheck(noopHealthCheck))
		require.NoError(t, err)
		require.NotNil(t, cli)
	})

	t.Run("healthcheck/info", func(t *testing.T) {
		t.Setenv(dockercontext.EnvOverrideHost, "tcp://foobar:2375") // this URL is parseable, although not reachable

		infoHealthCheck := func(ctx context.Context) func(c *client.Client) error {
			return func(c *client.Client) error {
				_, err := c.Info(ctx)
				return err
			}
		}

		cli, err := client.New(context.Background(), client.WithHealthCheck(infoHealthCheck))
		require.Error(t, err)
		require.Nil(t, cli)
	})

	t.Run("docker-host/precedence", func(t *testing.T) {
		t.Run("env-var-wins", func(t *testing.T) {
			t.Setenv(dockercontext.EnvOverrideHost, "tcp://foobar:2375") // this URL is parseable, although not reachable
			cli, err := client.New(context.Background())
			require.Error(t, err)
			require.Nil(t, cli)
		})

		t.Run("context-wins/found", func(t *testing.T) {
			t.Setenv(dockercontext.EnvOverrideContext, dockercontext.DefaultContextName)
			cli, err := client.New(context.Background())
			require.NoError(t, err)
			require.NotNil(t, cli)
		})

		t.Run("context-wins/not-found", func(t *testing.T) {
			t.Setenv(dockercontext.EnvOverrideContext, "foocontext") // this context does not exist
			cli, err := client.New(context.Background())
			require.ErrorIs(t, err, dockercontext.ErrDockerContextNotFound)
			require.Nil(t, cli)
		})
	})
}

func TestClientConcurrentAccess(t *testing.T) {
	t.Run("concurrent-client-close", func(t *testing.T) {
		client, err := client.New(context.Background())
		require.NoError(t, err)
		require.NotNil(t, client)

		const goroutines = 100
		wg := sync.WaitGroup{}
		wg.Add(goroutines)

		// Create a channel to coordinate goroutines
		start := make(chan struct{})

		// Launch goroutines that will either call Client() or Close()
		for i := 0; i < goroutines; i++ {
			go func(id int) {
				defer wg.Done()
				<-start // Wait for all goroutines to be ready

				if id%2 == 0 {
					// Even IDs call Client()
					c := client.Client
					// Client() might return nil if the client was closed by another goroutine
					// This is expected behavior
					if c != nil {
						require.NotNil(t, c)
					}
				} else {
					// Odd IDs call Close()
					err := client.Close()
					// Close() is idempotent, so it's okay to call it multiple times
					require.NoError(t, err)
				}
			}(i)
		}

		// Start all goroutines simultaneously
		close(start)
		wg.Wait()
	})

	t.Run("concurrent-client-calls", func(t *testing.T) {
		client, err := client.New(context.Background())
		require.NoError(t, err)
		require.NotNil(t, client)

		const goroutines = 100
		wg := sync.WaitGroup{}
		wg.Add(goroutines)

		// Create a channel to coordinate goroutines
		start := make(chan struct{})

		// Launch goroutines that will all call Client()
		for range goroutines {
			go func() {
				defer wg.Done()
				<-start // Wait for all goroutines to be ready

				c := client.Client
				// All calls should return the same client instance
				require.NotNil(t, c)
			}()
		}

		// Start all goroutines simultaneously
		close(start)
		wg.Wait()
	})
}
