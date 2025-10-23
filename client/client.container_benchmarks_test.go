package client_test

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"

	dockerclient "github.com/moby/moby/client"
	"github.com/stretchr/testify/require"

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
				_, err := dockerClient.ContainerList(context.Background(), dockerclient.ContainerListOptions{All: true})
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
		_, err = dockerClient.ContainerStart(context.Background(), containerName, dockerclient.ContainerStartOptions{})
		require.NoError(b, err)

		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := dockerClient.ContainerPause(context.Background(), containerName, dockerclient.ContainerPauseOptions{})
			require.NoError(b, err)

			_, err = dockerClient.ContainerUnpause(context.Background(), containerName, dockerclient.ContainerUnpauseOptions{})
			require.NoError(b, err)
		}
	})
}
