package image

import (
	"log/slog"
)

// ImageClient is a client to perform operations on images.
type ImageClient interface {
	// Close closes the client.
	Close() error

	// Logger returns the logger.
	Logger() *slog.Logger
}
