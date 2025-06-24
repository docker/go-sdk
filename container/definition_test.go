package container

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/docker/api/types/container"
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

	t.Run("same-source-multiple-targets", func(t *testing.T) {
		d := &Definition{
			hostConfigModifier: func(hc *container.HostConfig) {
				hc.Binds = []string{"/data:/srv", "/data:/data"}
			},
		}
		err := d.validateMounts()
		require.NoError(t, err)
	})

	t.Run("bind-options/provided", func(t *testing.T) {
		d := &Definition{
			hostConfigModifier: func(hc *container.HostConfig) {
				hc.Binds = []string{"/a:/a:nocopy", "/b:/b:ro", "/c:/c:rw", "/d:/d:z", "/e:/e:Z", "/f:/f:shared", "/g:/g:rshared", "/h:/h:slave", "/i:/i:rslave", "/j:/j:private", "/k:/k:rprivate", "/l:/l:ro,z,shared"}
			},
		}
		err := d.validateMounts()
		require.NoError(t, err)
	})
}
