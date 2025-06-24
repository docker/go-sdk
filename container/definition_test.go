package container

import (
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/require"
)

func TestValidateMounts(t *testing.T) {
	t.Run("no-host-config-modifier", func(t *testing.T) {
		d := &Definition{}
		err := d.validateMounts()
		require.NoError(t, err)
	})

	t.Run("invalid-bind-mount", func(t *testing.T) {
		d := &Definition{
			hostConfigModifier: func(hc *container.HostConfig) {
				hc.Binds = []string{"foo"}
			},
		}
		err := d.validateMounts()
		require.ErrorIs(t, err, ErrInvalidBindMount)
	})

	t.Run("duplicate-mount-target", func(t *testing.T) {
		d := &Definition{
			hostConfigModifier: func(hc *container.HostConfig) {
				hc.Binds = []string{"/foo:/duplicated", "/bar:/duplicated"}
			},
		}
		err := d.validateMounts()
		require.ErrorIs(t, err, ErrDuplicateMountTarget)
	})
}
