package client

import (
	"github.com/containerd/errdefs"
)

var permanentClientErrors = []func(error) bool{
	errdefs.IsNotFound,
	errdefs.IsInvalidArgument,
	errdefs.IsUnauthorized,
	errdefs.IsPermissionDenied,
	errdefs.IsNotImplemented,
	errdefs.IsInternal,
}

// IsPermanentClientError returns true if the error is a permanent client error.
func IsPermanentClientError(err error) bool {
	for _, isErrFn := range permanentClientErrors {
		if isErrFn(err) {
			return true
		}
	}
	return false
}
