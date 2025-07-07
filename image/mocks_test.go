package image

import (
	"bytes"
	"context"
	"io"
	"log/slog"

	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

// errMockCli is a mock implementation of client.APIClient, which is handy for simulating
// error returns in retry scenarios.
type errMockCli struct {
	client.APIClient

	err             error
	imageBuildCount int
	imagePullCount  int
	logger          *slog.Logger
}

func (f *errMockCli) ImageBuild(_ context.Context, _ build.ImageBuildOptions) (build.ImageBuildResponse, error) {
	f.imageBuildCount++

	// In real Docker API, the response body contains JSON build messages, not the build context
	// For testing purposes, we can return an empty JSON stream or some mock build output
	mockBuildOutput := `{"stream":"Step 1/1 : FROM hello-world"}
{"stream":"Successfully built abc123"}
`
	responseBody := io.NopCloser(bytes.NewBufferString(mockBuildOutput))
	return build.ImageBuildResponse{Body: responseBody}, f.err
}

func (f *errMockCli) ImagePull(_ context.Context, _ string, _ image.PullOptions) (io.ReadCloser, error) {
	f.imagePullCount++
	return io.NopCloser(&bytes.Buffer{}), f.err
}

func (f *errMockCli) Close() error {
	return nil
}

func (f *errMockCli) Logger() *slog.Logger {
	if f.logger == nil {
		f.logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	return f.logger
}
