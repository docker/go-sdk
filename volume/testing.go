package volume

import (
	"context"
	"reflect"
	"regexp"
	"testing"

	"github.com/containerd/errdefs"
	"github.com/stretchr/testify/require"

	"github.com/docker/docker/api/types/volume"
	"github.com/docker/go-sdk/client"
)

// errAlreadyInProgress is a regular expression that matches the error for a volume
// removal that is already in progress.
var errAlreadyInProgress = regexp.MustCompile(`removal of volume .* is already in progress`)

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

// Cleanup is a helper function that schedules the volume to be
// removed when the test ends.
// This should be the first call after [New] in a test before
// any error check. If volume is nil, it's a no-op.
func Cleanup(tb testing.TB, v TerminableVolume) {
	tb.Helper()

	tb.Cleanup(func() {
		if !isNil(v) {
			noErrorOrIgnored(tb, v.Terminate(context.Background()))
		}
	})
}

// CleanupByID is a helper function that schedules the volume to be
// removed, identified by its ID, when the test ends.
// This should be the first call after New(...) in a test before
// any error check. If volume is nil, it's a no-op.
// It uses a new docker client to terminate the volume, which is automatically
// closed when the test ends.
func CleanupByID(tb testing.TB, id string) {
	tb.Helper()

	dockerClient, err := client.New(context.Background())
	if err != nil {
		noErrorOrIgnored(tb, err)
	}

	// synthetic network using a new docker client.
	nw := &Volume{
		Volume: &volume.Volume{
			Name: id,
		},
		dockerClient: dockerClient,
	}
	tb.Cleanup(func() {
		noErrorOrIgnored(tb, dockerClient.Close())
	})

	Cleanup(tb, nw)
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

// isNil returns true if val is nil or a nil instance false otherwise.
func isNil(val any) bool {
	if val == nil {
		return true
	}

	valueOf := reflect.ValueOf(val)
	switch valueOf.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return valueOf.IsNil()
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
