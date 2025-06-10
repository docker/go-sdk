package exec

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/docker/pkg/stdcopy"
)

func TestSafeBuffer(t *testing.T) {
	t.Run("basic-write-and-read", func(t *testing.T) {
		sb := &safeBuffer{}
		data := []byte("test data")

		// Write data
		n, err := sb.Write(data)
		require.NoError(t, err)
		require.Equal(t, len(data), n)

		// Read data
		buf := make([]byte, len(data))
		n, err = sb.Read(buf)
		require.NoError(t, err)
		require.Equal(t, len(data), n)
		require.Equal(t, data, buf)
	})

	t.Run("error-propagation", func(t *testing.T) {
		sb := &safeBuffer{}
		testErr := errors.New("test error")

		// Set error
		sb.Error(testErr)

		// Try to read
		buf := make([]byte, 10)
		_, err := sb.Read(buf)
		require.Equal(t, testErr, err)
	})

	t.Run("concurrent-write-and-read", func(t *testing.T) {
		sb := &safeBuffer{}
		var wg sync.WaitGroup

		// Start multiple writers
		for range 10 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				data := []byte("test data")
				_, err := sb.Write(data)
				require.NoError(t, err)
			}()
		}

		// Start multiple readers
		for range 10 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				buf := make([]byte, 9) // "test data" length
				_, err := sb.Read(buf)
				if err != nil && !errors.Is(err, io.EOF) {
					require.NoError(t, err)
				}
			}()
		}

		wg.Wait()
	})

	t.Run("read-empty-buffer", func(t *testing.T) {
		sb := &safeBuffer{}
		buf := make([]byte, 10)
		_, err := sb.Read(buf)
		require.Equal(t, io.EOF, err)
	})

	t.Run("write-after-error", func(t *testing.T) {
		sb := &safeBuffer{}
		testErr := errors.New("test error")
		sb.Error(testErr)

		// Try to write after error
		_, err := sb.Write([]byte("test"))
		require.NoError(t, err) // Write should still work

		// Read should still return the error
		buf := make([]byte, 10)
		_, err = sb.Read(buf)
		require.Equal(t, testErr, err)
	})
}

func TestNewProcessOptions(t *testing.T) {
	t.Run("default-values", func(t *testing.T) {
		cmd := []string{"echo", "hello"}
		opts := NewProcessOptions(cmd)

		require.NotNil(t, opts)
		require.Equal(t, cmd, opts.ExecConfig.Cmd)
		require.False(t, opts.ExecConfig.Detach)
		require.True(t, opts.ExecConfig.AttachStdout)
		require.True(t, opts.ExecConfig.AttachStderr)
		require.Empty(t, opts.ExecConfig.User)
		require.Empty(t, opts.ExecConfig.WorkingDir)
		require.Empty(t, opts.ExecConfig.Env)
		require.False(t, opts.ExecConfig.Tty)
		require.Nil(t, opts.Reader)
	})
}

func TestProcessOptions(t *testing.T) {
	t.Run("WithUser", func(t *testing.T) {
		opts := NewProcessOptions([]string{"echo"})
		WithUser("testuser").Apply(opts)
		require.Equal(t, "testuser", opts.ExecConfig.User)
	})

	t.Run("WithWorkingDir", func(t *testing.T) {
		opts := NewProcessOptions([]string{"echo"})
		WithWorkingDir("/test/dir").Apply(opts)
		require.Equal(t, "/test/dir", opts.ExecConfig.WorkingDir)
	})

	t.Run("WithEnv", func(t *testing.T) {
		opts := NewProcessOptions([]string{"echo"})
		env := []string{"TEST=value", "FOO=bar"}
		WithEnv(env).Apply(opts)
		require.Equal(t, env, opts.ExecConfig.Env)
	})

	t.Run("WithTTY", func(t *testing.T) {
		opts := NewProcessOptions([]string{"echo"})
		WithTTY(true).Apply(opts)
		require.True(t, opts.ExecConfig.Tty)
	})

	t.Run("multiple-options", func(t *testing.T) {
		opts := NewProcessOptions([]string{"echo"})

		// Apply multiple options
		WithUser("testuser").Apply(opts)
		WithWorkingDir("/test/dir").Apply(opts)
		WithEnv([]string{"TEST=value"}).Apply(opts)
		WithTTY(true).Apply(opts)

		// Verify all options were applied
		require.Equal(t, "testuser", opts.ExecConfig.User)
		require.Equal(t, "/test/dir", opts.ExecConfig.WorkingDir)
		require.Equal(t, []string{"TEST=value"}, opts.ExecConfig.Env)
		require.True(t, opts.ExecConfig.Tty)
	})

	t.Run("default-values-not-affected", func(t *testing.T) {
		opts := NewProcessOptions([]string{"echo"})

		// Apply options
		WithUser("testuser").Apply(opts)
		WithTTY(true).Apply(opts)

		// Verify defaults are still set
		require.False(t, opts.ExecConfig.Detach)
		require.True(t, opts.ExecConfig.AttachStdout)
		require.True(t, opts.ExecConfig.AttachStderr)
	})
}

func TestProcessOptionFunc(t *testing.T) {
	t.Run("implements-ProcessOption-interface", func(_ *testing.T) {
		var _ ProcessOption = ProcessOptionFunc(func(_ *ProcessOptions) {})
	})

	t.Run("applies-function-to-options", func(t *testing.T) {
		opts := NewProcessOptions([]string{"echo"})
		called := false

		fn := ProcessOptionFunc(func(opts *ProcessOptions) {
			called = true
			opts.ExecConfig.User = "testuser"
		})

		fn.Apply(opts)
		require.True(t, called)
		require.Equal(t, "testuser", opts.ExecConfig.User)
	})
}

func TestMultiplexed(t *testing.T) {
	t.Run("nil-reader", func(t *testing.T) {
		opts := NewProcessOptions([]string{"echo"})
		Multiplexed().Apply(opts)
		require.Nil(t, opts.Reader, "Reader should remain nil when no reader is set")
	})

	t.Run("combines-stdout-and-stderr", func(t *testing.T) {
		var buf bytes.Buffer
		writer := stdcopy.NewStdWriter(&buf, stdcopy.Stdout)
		_, err := writer.Write([]byte("stdout output"))
		require.NoError(t, err)
		writer = stdcopy.NewStdWriter(&buf, stdcopy.Stderr)
		_, err = writer.Write([]byte("stderr output"))
		require.NoError(t, err)

		opts := NewProcessOptions([]string{"echo"})
		opts.Reader = &buf
		Multiplexed().Apply(opts)

		output, err := io.ReadAll(opts.Reader)
		require.NoError(t, err)
		outputStr := string(output)

		require.Contains(t, outputStr, "stdout output")
		require.Contains(t, outputStr, "stderr output")
	})

	t.Run("empty-output", func(t *testing.T) {
		opts := NewProcessOptions([]string{"echo"})
		opts.Reader = bytes.NewReader(nil)
		Multiplexed().Apply(opts)

		output, err := io.ReadAll(opts.Reader)
		require.NoError(t, err)
		require.Empty(t, output)
	})

	t.Run("read-error", func(t *testing.T) {
		errReader := &errorReader{err: errors.New("read error")}

		opts := NewProcessOptions([]string{"echo"})
		opts.Reader = errReader
		Multiplexed().Apply(opts)

		buf := make([]byte, 10)
		_, err := opts.Reader.Read(buf)
		require.Error(t, err)
		require.Contains(t, err.Error(), "copying output")
	})

	t.Run("partial-read", func(t *testing.T) {
		var buf bytes.Buffer
		writer := stdcopy.NewStdWriter(&buf, stdcopy.Stdout)
		_, err := writer.Write([]byte("stdout output"))
		require.NoError(t, err)
		writer = stdcopy.NewStdWriter(&buf, stdcopy.Stderr)
		_, err = writer.Write([]byte("stderr output"))
		require.NoError(t, err)

		opts := NewProcessOptions([]string{"echo"})
		opts.Reader = &buf
		Multiplexed().Apply(opts)

		// Read in small chunks
		smallBuf := make([]byte, 5)
		var output strings.Builder
		for {
			n, err := opts.Reader.Read(smallBuf)
			if err == io.EOF {
				break
			}
			require.NoError(t, err)
			output.Write(smallBuf[:n])
		}

		outputStr := output.String()
		require.Contains(t, outputStr, "stdout output")
		require.Contains(t, outputStr, "stderr output")
	})

	t.Run("large-output", func(t *testing.T) {
		largeOutput := strings.Repeat("x", 1024*1024) // 1MB of data

		var buf bytes.Buffer
		writer := stdcopy.NewStdWriter(&buf, stdcopy.Stdout)
		_, err := writer.Write([]byte(largeOutput))
		require.NoError(t, err)
		writer = stdcopy.NewStdWriter(&buf, stdcopy.Stderr)
		_, err = writer.Write([]byte(largeOutput))
		require.NoError(t, err)

		opts := NewProcessOptions([]string{"echo"})
		opts.Reader = &buf
		Multiplexed().Apply(opts)

		output, err := io.ReadAll(opts.Reader)
		require.NoError(t, err)
		require.Len(t, output, len(largeOutput)*2)
	})
}

// errorReader is a reader that always returns an error
type errorReader struct {
	err error
}

func (r *errorReader) Read(_ []byte) (n int, err error) {
	return 0, r.err
}
