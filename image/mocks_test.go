package image

import (
	"bytes"
	"context"
	"io"
	"iter"

	"github.com/moby/moby/api/types/jsonstream"
	"github.com/moby/moby/client"
)

// errMockCli is a mock implementation of client.APIClient, which is handy for simulating
// error returns in retry scenarios.
type errMockCli struct {
	client.APIClient

	err             error
	imageBuildCount int
	imagePullCount  int
}

func (f *errMockCli) Ping(_ context.Context, _ client.PingOptions) (client.PingResult, error) {
	return client.PingResult{}, nil
}

func (f *errMockCli) ImageBuild(_ context.Context, _ io.Reader, _ client.ImageBuildOptions) (client.ImageBuildResult, error) {
	f.imageBuildCount++

	// In real Docker API, the response body contains JSON build messages, not the build context
	// For testing purposes, we can return an empty JSON stream or some mock build output
	mockBuildOutput := `{"stream":"Step 1/1 : FROM hello-world"}
{"stream":"Successfully built abc123"}
`
	responseBody := io.NopCloser(bytes.NewBufferString(mockBuildOutput))
	return client.ImageBuildResult{Body: responseBody}, f.err
}

func (f *errMockCli) ImagePull(_ context.Context, _ string, _ client.ImagePullOptions) (client.ImagePullResponse, error) {
	f.imagePullCount++
	// Return mock JSON messages similar to real Docker pull output
	mockPullOutput := `{"status":"Pulling from library/nginx","id":"latest"}
{"status":"Pull complete","id":"abc123"}
`
	return errMockImagePullResponse{ReadCloser: io.NopCloser(bytes.NewBufferString(mockPullOutput))}, f.err
}

func (f *errMockCli) Close() error {
	return nil
}

type errMockImagePullResponse struct {
	io.ReadCloser
}

func (f errMockImagePullResponse) JSONMessages(_ context.Context) iter.Seq2[jsonstream.Message, error] {
	return nil
}

func (f errMockImagePullResponse) Wait(_ context.Context) error {
	return nil
}
