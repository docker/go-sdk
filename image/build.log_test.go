package image

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/docker/pkg/jsonmessage"
)

func TestLoggerWriter_Write(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		t.Run("json-message-with-error", func(t *testing.T) {
			testLoggerWriterWithJSONError(t)
		})

		t.Run("json-message-with-stream", func(t *testing.T) {
			testLoggerWriterWithJSONStream(t)
		})

		t.Run("json-message-with-status", func(t *testing.T) {
			testLoggerWriterWithJSONStatus(t)
		})

		t.Run("plain-text-message", func(t *testing.T) {
			testLoggerWriterWithPlainText(t)
		})

		t.Run("empty-message", func(t *testing.T) {
			testLoggerWriterWithEmptyMessage(t)
		})

		t.Run("invalid-json-fallback", func(t *testing.T) {
			testLoggerWriterWithInvalidJSON(t)
		})

		t.Run("stream-with-newline", func(t *testing.T) {
			testLoggerWriterStreamWithNewline(t)
		})

		t.Run("status-with-progress", func(t *testing.T) {
			testLoggerWriterStatusWithProgress(t)
		})
	})
}

func testLoggerWriterWithJSONError(t *testing.T) {
	t.Helper()

	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, nil))
	writer := &loggerWriter{logger: logger}

	errorMsg := jsonmessage.JSONMessage{
		Error: &jsonmessage.JSONError{
			Message: "build failed",
		},
	}
	jsonData, err := json.Marshal(errorMsg)
	require.NoError(t, err)

	n, err := writer.Write(jsonData)
	require.NoError(t, err)
	require.Equal(t, len(jsonData), n)

	logOutput := buf.String()
	require.Contains(t, logOutput, "Build error")
	require.Contains(t, logOutput, "build failed")
	require.Contains(t, logOutput, "level=ERROR")
}

func testLoggerWriterWithJSONStream(t *testing.T) {
	t.Helper()

	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, nil))
	writer := &loggerWriter{logger: logger}

	streamMsg := jsonmessage.JSONMessage{
		Stream: "Step 1/3 : FROM ubuntu:latest",
	}
	jsonData, err := json.Marshal(streamMsg)
	require.NoError(t, err)

	n, err := writer.Write(jsonData)
	require.NoError(t, err)
	require.Equal(t, len(jsonData), n)

	logOutput := buf.String()
	require.Contains(t, logOutput, "Step 1/3 : FROM ubuntu:latest")
	require.Contains(t, logOutput, "level=INFO")
}

func testLoggerWriterWithJSONStatus(t *testing.T) {
	t.Helper()

	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, nil))
	writer := &loggerWriter{logger: logger}

	statusMsg := jsonmessage.JSONMessage{
		Status: "Downloading",
		ID:     "abc123",
	}
	jsonData, err := json.Marshal(statusMsg)
	require.NoError(t, err)

	n, err := writer.Write(jsonData)
	require.NoError(t, err)
	require.Equal(t, len(jsonData), n)

	logOutput := buf.String()
	require.Contains(t, logOutput, "Downloading")
	require.Contains(t, logOutput, "id=abc123")
	require.Contains(t, logOutput, "level=INFO")
}

func testLoggerWriterWithPlainText(t *testing.T) {
	t.Helper()

	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, nil))
	writer := &loggerWriter{logger: logger}

	plainText := "This is a plain text message"

	n, err := writer.Write([]byte(plainText))
	require.NoError(t, err)
	require.Equal(t, len(plainText), n)

	logOutput := buf.String()
	require.Contains(t, logOutput, plainText)
	require.Contains(t, logOutput, "level=INFO")
}

func testLoggerWriterWithEmptyMessage(t *testing.T) {
	t.Helper()

	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, nil))
	writer := &loggerWriter{logger: logger}

	n, err := writer.Write([]byte(""))
	require.NoError(t, err)
	require.Equal(t, 0, n)

	logOutput := buf.String()
	require.Empty(t, logOutput)
}

func testLoggerWriterWithInvalidJSON(t *testing.T) {
	t.Helper()

	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, nil))
	writer := &loggerWriter{logger: logger}

	invalidJSON := "{ invalid json"

	n, err := writer.Write([]byte(invalidJSON))
	require.NoError(t, err)
	require.Equal(t, len(invalidJSON), n)

	logOutput := buf.String()
	require.Contains(t, logOutput, invalidJSON)
	require.Contains(t, logOutput, "level=INFO")
}

func testLoggerWriterStreamWithNewline(t *testing.T) {
	t.Helper()

	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, nil))
	writer := &loggerWriter{logger: logger}

	streamMsg := jsonmessage.JSONMessage{
		Stream: "Step 1/3 : FROM ubuntu:latest\n",
	}
	jsonData, err := json.Marshal(streamMsg)
	require.NoError(t, err)

	n, err := writer.Write(jsonData)
	require.NoError(t, err)
	require.Equal(t, len(jsonData), n)

	logOutput := buf.String()
	require.Contains(t, logOutput, "Step 1/3 : FROM ubuntu:latest")
	require.NotContains(t, logOutput, "Step 1/3 : FROM ubuntu:latest\n")
	require.Contains(t, logOutput, "level=INFO")
}

func testLoggerWriterStatusWithProgress(t *testing.T) {
	t.Helper()

	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, nil))
	writer := &loggerWriter{logger: logger}

	statusMsg := jsonmessage.JSONMessage{
		Status:   "Downloading",
		ID:       "abc123",
		Progress: &jsonmessage.JSONProgress{Current: 1024, Total: 2048},
	}
	jsonData, err := json.Marshal(statusMsg)
	require.NoError(t, err)

	n, err := writer.Write(jsonData)
	require.NoError(t, err)
	require.Equal(t, len(jsonData), n)

	logOutput := buf.String()
	require.Contains(t, logOutput, "Downloading")
	require.Contains(t, logOutput, "id=abc123")
	require.Contains(t, logOutput, "progress=")
	require.Contains(t, logOutput, "level=INFO")
}

func TestLoggerWriter_Write_EdgeCases(t *testing.T) {
	t.Run("whitespace-only-message", func(t *testing.T) {
		testLoggerWriterWithWhitespaceOnly(t)
	})

	t.Run("newline-only-message", func(t *testing.T) {
		testLoggerWriterWithNewlineOnly(t)
	})

	t.Run("json-message-with-empty-fields", func(t *testing.T) {
		testLoggerWriterWithEmptyJSONFields(t)
	})
}

func testLoggerWriterWithWhitespaceOnly(t *testing.T) {
	t.Helper()

	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, nil))
	writer := &loggerWriter{logger: logger}

	whitespace := "   \t   "

	n, err := writer.Write([]byte(whitespace))
	require.NoError(t, err)
	require.Equal(t, len(whitespace), n)

	logOutput := buf.String()
	require.Contains(t, logOutput, strings.TrimSpace(whitespace))
	require.Contains(t, logOutput, "level=INFO")
}

func testLoggerWriterWithNewlineOnly(t *testing.T) {
	t.Helper()

	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, nil))
	writer := &loggerWriter{logger: logger}

	newlineOnly := "\n"

	n, err := writer.Write([]byte(newlineOnly))
	require.NoError(t, err)
	require.Equal(t, len(newlineOnly), n)

	logOutput := buf.String()
	require.Empty(t, logOutput)
}

func testLoggerWriterWithEmptyJSONFields(t *testing.T) {
	t.Helper()

	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, nil))
	writer := &loggerWriter{logger: logger}

	emptyMsg := jsonmessage.JSONMessage{
		Stream: "",
		Status: "",
		ID:     "",
	}
	jsonData, err := json.Marshal(emptyMsg)
	require.NoError(t, err)

	n, err := writer.Write(jsonData)
	require.NoError(t, err)
	require.Equal(t, len(jsonData), n)

	// Should not log anything since all fields are empty
	logOutput := buf.String()
	require.Empty(t, logOutput)
}
