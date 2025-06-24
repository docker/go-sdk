package client_test

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/docker/go-sdk/client"
)

func BenchmarkContainerList(b *testing.B) {
	dockerClient, err := client.New(context.Background())
	require.NoError(b, err)
	require.NotNil(b, dockerClient)

	pullImage(b, dockerClient, "nginx:alpine")

	max := 5

	wg := sync.WaitGroup{}
	wg.Add(max)

	for i := range max {
		go func(i int) {
			defer wg.Done()

			resp, err := dockerClient.ContainerCreate(context.Background(), &container.Config{
				Image: "nginx:alpine",
				ExposedPorts: nat.PortSet{
					"80/tcp": {},
				},
			}, nil, nil, nil, fmt.Sprintf("nginx-test-name-%d", i))
			require.NoError(b, err)
			require.NotNil(b, resp)
			require.NotEmpty(b, resp.ID)

			b.Cleanup(func() {
				err := dockerClient.ContainerRemove(context.Background(), resp.ID, container.RemoveOptions{})
				require.NoError(b, err)
			})
		}(i)
	}

	wg.Wait()

	b.Run("container-list", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, err := dockerClient.ContainerList(context.Background(), container.ListOptions{All: true})
				require.NoError(b, err)
			}
		})
	})

	b.Run("find-container-by-name", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, err := dockerClient.FindContainerByName(context.Background(), fmt.Sprintf("nginx-test-name-%d", rand.Intn(max)))
				require.NoError(b, err)
			}
		})
	})
}
