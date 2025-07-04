package image_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"path"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/client"
	"github.com/docker/go-sdk/image"
)

var buildPath = path.Join("testdata", "build")

func BenchmarkBuild(b *testing.B) {
	b.Run("success", func(b *testing.B) {
		contextArchive, err := image.ArchiveBuildContext(buildPath, "Dockerfile")
		require.NoError(b, err)

		// Buffer the entire archive data
		archiveData, err := io.ReadAll(contextArchive)
		require.NoError(b, err)

		bInfo := &testBuildInfo{
			// using a log writer to avoid writing to stdout, dirtying the benchmark output
			logWriter: &bytes.Buffer{},
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := range b.N {
			// Create fresh reader from buffered data
			bInfo.contextArchive = bytes.NewReader(archiveData)
			// Use a unique tag for each iteration to avoid collisions
			bInfo.imageTag = fmt.Sprintf("test:benchmark-%d", i)
			testBuild(b, bInfo)
		}
	})

	b.Run("from-dir", func(b *testing.B) {
		buildPath := path.Join("testdata", "build")

		// using a buffer to capture the build output
		buf := &bytes.Buffer{}
		logger := slog.New(slog.NewTextHandler(buf, nil))
		cli, err := client.New(context.Background(), client.WithLogger(logger))
		require.NoError(b, err)

		b.ResetTimer()
		b.ReportAllocs()

		for i := range b.N {
			// Use a unique tag for each iteration to avoid collisions
			tag := fmt.Sprintf("test:benchmark-%d", i)
			_, err := image.BuildFromDir(context.Background(), buildPath, "Dockerfile", tag, image.WithBuildClient(cli))
			require.NoError(b, err)

			b.Cleanup(func() {
				cleanup(b, tag)
			})
		}
	})
}
