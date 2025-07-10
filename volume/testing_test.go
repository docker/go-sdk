package volume_test

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/volume"
)

func TestCleanup(t *testing.T) {
	t.Run("cleanup-by-id-nonexistent", func(t *testing.T) {
		// Tests ErrNotFound case - should be cleanup safe
		volume.CleanupByID(t, "nonexistent-volume-id")
	})

	t.Run("cleanup-nil-volume", func(t *testing.T) {
		// Tests nil case - should be cleanup safe
		volume.Cleanup(t, nil)
	})

	t.Run("concurrent-cleanups", func(t *testing.T) {
		// Tests ErrNotFound case after first cleanup
		v, err := volume.New(context.Background())
		require.NoError(t, err)

		// ten goroutines trying to cleanup the same volume
		wg := sync.WaitGroup{}
		wg.Add(50)
		for i := 0; i < 50; i++ {
			go func() {
				volume.Cleanup(t, v)
				wg.Done()
			}()
		}
		wg.Wait()
	})
}
