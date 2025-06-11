package dockercontainer

import (
	"regexp"
	"testing"

	"github.com/containerd/errdefs"
	"github.com/stretchr/testify/require"
)

// errAlreadyInProgress is a regular expression that matches the error for a container
// removal that is already in progress.
var errAlreadyInProgress = regexp.MustCompile(`removal of container .* is already in progress`)

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

// CleanupContainer is a helper function that schedules a [TerminableContainer]
// to be terminated when the test ends.
//
// This should be called directly after (before any error check)
// [Create](...) in a test to ensure the
// container is pruned when the function ends.
// If the container is nil, it's a no-op.
func CleanupContainer(tb testing.TB, ctr TerminableContainer, options ...TerminateOption) {
	tb.Helper()

	tb.Cleanup(func() {
		noErrorOrIgnored(tb, TerminateContainer(ctr, options...))
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
