package volume

import (
	"context"
	"fmt"
)

// TerminableVolume is a volume that can be terminated.
type TerminableVolume interface {
	Terminate(ctx context.Context, opts ...TerminateOption) error
}

// Terminate terminates the volume.
func (v *Volume) Terminate(ctx context.Context, opts ...TerminateOption) error {
	terminateOptions := &terminateOptions{}
	for _, opt := range opts {
		if err := opt(terminateOptions); err != nil {
			return fmt.Errorf("apply option: %w", err)
		}
	}

	return v.dockerClient.VolumeRemove(ctx, v.Name, terminateOptions.force)
}
