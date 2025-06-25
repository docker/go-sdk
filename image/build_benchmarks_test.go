package image_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/go-sdk/client"
	"github.com/docker/go-sdk/image"
)

var buildFiles = []image.BuildFile{
	{
		Name:    "say_hi.sh",
		Content: []byte(`echo hi this is from the say_hi.sh file!`),
	},
	{
		Name: "Dockerfile",
		Content: []byte(`FROM alpine
				WORKDIR /app
				COPY . .
				CMD ["sh", "./say_hi.sh"]`),
	},
}

func BenchmarkBuild(b *testing.B) {
	b.Run("success", func(b *testing.B) {
		contextArchive, err := image.ReaderFromFiles(buildFiles)
		require.NoError(b, err)

		bInfo := &testBuildInfo{
			contextArchive: contextArchive,
			logWriter:      &bytes.Buffer{},
			imageTag:       "test:test",
		}

		b.ResetTimer()
		b.ReportAllocs()

		for range b.N {
			testBuild(b, bInfo)
		}
	})

	b.Cleanup(func() {
		cli, err := client.New(context.Background())
		require.NoError(b, err)

		containers, err := cli.ContainerList(context.Background(), container.ListOptions{
			Filters: filters.NewArgs(filters.Arg("label", fmt.Sprintf("%s=%s", labelImageBuildTestKey, labelImageBuildTestValue))),
			All:     true,
		})
		require.NoError(b, err)

		for _, ctr := range containers {
			require.NoError(b, cli.ContainerRemove(context.Background(), ctr.ID, container.RemoveOptions{Force: true}))
		}
	})
}
