package dockernetwork

import (
	"context"
	"regexp"
	"testing"

	"github.com/containerd/errdefs"
	"github.com/stretchr/testify/require"
)

// errAlreadyInProgress is a regular expression that matches the error for a network
// removal that is already in progress.
var errAlreadyInProgress = regexp.MustCompile(`removal of network .* is already in progress`)

// causer is an interface that allows to get the cause of an error.
type causer interface {
	Cause() error
}

// wrapErr is an interface that allows to unwrap an error.
type wrapErr interface {
	Unwrap() error
}

// unwrapErrs is an interface that allows to unwrap multiple errors.
type unwrapErrs interface {
	Unwrap() []error
}

// CleanupNetwork is a helper function that schedules the network to be
// removed when the test ends.
// This should be the first call after NewNetwork(...) in a test before
// any error check. If network is nil, it's a no-op.
func CleanupNetwork(tb testing.TB, nw TerminableNetwork) {
	tb.Helper()

	tb.Cleanup(func() {
		if !isNil(nw) {
			noErrorOrIgnored(tb, nw.Terminate(context.Background()))
		}
	})
}

// isCleanupSafe checks if an error is cleanup safe.
func isCleanupSafe(err error) bool {
	if err == nil {
		return true
	}

	// First try with containerd's errdefs
	switch {
	case errdefs.IsNotFound(err):
		return true
	case errdefs.IsConflict(err):
		// Terminating a container that is already terminating.
		if errAlreadyInProgress.MatchString(err.Error()) {
			return true
		}
		return false
	}

	switch x := err.(type) { //nolint:errorlint // We need to check for interfaces.
	case causer:
		return isCleanupSafe(x.Cause())
	case wrapErr:
		return isCleanupSafe(x.Unwrap())
	case unwrapErrs:
		for _, e := range x.Unwrap() {
			if !isCleanupSafe(e) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// noErrorOrIgnored is a helper function that checks if the error is nil or an error
// we can ignore.
func noErrorOrIgnored(tb testing.TB, err error) {
	tb.Helper()

	if isCleanupSafe(err) {
		return
	}

	require.NoError(tb, err)
}
