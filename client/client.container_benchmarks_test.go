package client_test

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-sdk/client"
)

func BenchmarkContainerList(b *testing.B) {
	dockerClient, err := client.New(context.Background())
	require.NoError(b, err)
	require.NotNil(b, dockerClient)

	img := "nginx:alpine"

	pullImage(b, dockerClient, img)

	max := 5

	wg := sync.WaitGroup{}
	wg.Add(max)

	for i := range max {
		go func(i int) {
			defer wg.Done()

			createContainer(b, dockerClient, img, fmt.Sprintf("nginx-test-name-%d", i))
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

func BenchmarkContainerPause(b *testing.B) {
	dockerClient, err := client.New(context.Background())
	require.NoError(b, err)
	require.NotNil(b, dockerClient)

	img := "nginx:alpine"

	containerName := "nginx-test-pause"

	pullImage(b, dockerClient, img)
	createContainer(b, dockerClient, img, containerName)

	b.Run("container-pause-unpause", func(b *testing.B) {
		err = dockerClient.ContainerStart(context.Background(), containerName, container.StartOptions{})
		require.NoError(b, err)

		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			err := dockerClient.ContainerPause(context.Background(), containerName)
			require.NoError(b, err)

			err = dockerClient.ContainerUnpause(context.Background(), containerName)
			require.NoError(b, err)
		}
	})
}
