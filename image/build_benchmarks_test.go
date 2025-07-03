package image_test

import (
	"bytes"
	"fmt"
	"io"
	"path"
	"testing"

	"github.com/stretchr/testify/require"

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
}
