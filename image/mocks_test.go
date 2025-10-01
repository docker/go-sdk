package image

import (
	"bytes"
	"context"
	"io"

	"github.com/docker/docker/api/types"
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
}

func (f *errMockCli) Ping(_ context.Context) (types.Ping, error) {
	return types.Ping{}, nil
}

func (f *errMockCli) ImageBuild(_ context.Context, _ io.Reader, _ build.ImageBuildOptions) (build.ImageBuildResponse, error) {
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
