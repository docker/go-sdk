package image

import (
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/docker/docker/pkg/jsonmessage"
)

// loggerWriter is a custom writer that forwards to the slog.Logger
type loggerWriter struct {
	logger *slog.Logger
}

// Write writes the message to the logger.
func (lw *loggerWriter) Write(p []byte) (int, error) {
	// Try to parse as JSON message first
	var msg jsonmessage.JSONMessage
	if err := json.Unmarshal(p, &msg); err == nil {
		// It's a JSON message, log it structured and there is no default case because
		// empty JSON messages should not be logged, to avoid noise.
		switch {
		case msg.Error != nil:
			lw.logger.Error("Build error", "error", msg.Error.Message)
		case msg.Stream != "":
			lw.logger.Info(strings.TrimSuffix(msg.Stream, "\n"))
		case msg.Status != "":
			lw.logger.Info(msg.Status, "id", msg.ID, "progress", msg.Progress)
		}
	} else {
		// Fall back to plain text
		text := strings.TrimSuffix(string(p), "\n")
		if text != "" {
			lw.logger.Info(text)
		}
	}
	return len(p), nil
}
