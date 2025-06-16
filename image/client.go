package image

import (
	"log/slog"
)

// ImageClient is a client to perform operations on images.
type ImageClient interface {
	// Logger returns the logger.
	Logger() *slog.Logger
}
