package wait_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/containerd/errdefs"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-sdk/container/wait"
)

const testFilename = "/tmp/file"

var anyContext = mock.MatchedBy(func(_ context.Context) bool { return true })

// newRunningTarget creates a new mockStrategyTarget that is running.
func newRunningTarget() *mockStrategyTarget {
	target := &mockStrategyTarget{}
	target.EXPECT().State(anyContext).
		Return(&container.State{Running: true}, nil)

	return target
}

// testForFile creates a new FileStrategy for testing.
func testForFile() *wait.FileStrategy {
	return wait.ForFile(testFilename).
		WithTimeout(time.Millisecond * 50).
		WithPollInterval(time.Millisecond)
}

func TestForFile(t *testing.T) {
	errNotFound := errdefs.ErrNotFound.WithMessage("file not found")
	ctx := context.Background()

	t.Run("not-found", func(t *testing.T) {
		target := newRunningTarget()
		target.EXPECT().CopyFromContainer(anyContext, testFilename).Return(nil, errNotFound)
		err := testForFile().WaitUntilReady(ctx, target)
		require.EqualError(t, err, context.DeadlineExceeded.Error())
	})

	t.Run("other-error", func(t *testing.T) {
		otherErr := errors.New("other error")
		target := newRunningTarget()
		target.EXPECT().CopyFromContainer(anyContext, testFilename).Return(nil, otherErr)
		err := testForFile().WaitUntilReady(ctx, target)
		require.ErrorIs(t, err, otherErr)
	})

	t.Run("valid", func(t *testing.T) {
		data := "my content\nwibble"
		file := bytes.NewBufferString(data)
		target := newRunningTarget()
		target.EXPECT().CopyFromContainer(anyContext, testFilename).Once().Return(nil, errNotFound)
		target.EXPECT().CopyFromContainer(anyContext, testFilename).Return(io.NopCloser(file), nil)
		var out bytes.Buffer
		err := testForFile().WithMatcher(func(r io.Reader) error {
			if _, err := io.Copy(&out, r); err != nil {
				return fmt.Errorf("copy: %w", err)
			}
			return nil
		}).WaitUntilReady(ctx, target)
		require.NoError(t, err)
		require.Equal(t, data, out.String())
	})
}
