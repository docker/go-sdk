package dockernetwork

import (
	"context"
	"errors"
	"fmt"
	"reflect"
)

// TerminableNetwork is a network that can be terminated.
type TerminableNetwork interface {
	Terminate(ctx context.Context) error
}

// Terminate is used to remove the network. It is usually triggered by as defer function.
func (n *Network) Terminate(ctx context.Context) error {
	if n.dockerClient == nil {
		return errors.New("docker client is not initialized")
	}

	if err := n.dockerClient.NetworkRemove(ctx, n.ID()); err != nil {
		return fmt.Errorf("terminate network: %w", err)
	}

	return nil
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
