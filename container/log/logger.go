package log

import (
	"log"
	"os"
)

// Validate our types implement the required interfaces.
var (
	_ Logger = (*log.Logger)(nil)
	_ Logger = (*noopLogger)(nil)
)

// Logger defines the Logger interface.
type Logger interface {
	Printf(format string, v ...any)
}

// defaultLogger is the default Logger instance.
var defaultLogger Logger = log.New(os.Stderr, "", log.LstdFlags)

// NoopLogger is a Logger that does nothing.
var NoopLogger Logger = &noopLogger{}

// Default returns the default Logger instance.
func Default() Logger {
	return defaultLogger
}

// SetDefault sets the default Logger instance.
func SetDefault(l Logger) {
	defaultLogger = l
}

// Printf implements Logging.
func Printf(format string, v ...any) {
	defaultLogger.Printf(format, v...)
}

type noopLogger struct{}

// Printf implements Logging.
func (n noopLogger) Printf(_ string, _ ...any) {
	// NOOP
}
