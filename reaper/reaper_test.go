package reaper_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/moby/moby/client"
	"github.com/stretchr/testify/require"

	sdkclient "github.com/docker/go-sdk/client"
	"github.com/docker/go-sdk/reaper"
)

func TestReaper_Integration(t *testing.T) {
	// Attempt to create a real client. Skip if daemon is not running.
	c, err := sdkclient.New(context.Background())
	if err != nil {
		t.Skip("Skipping integration test since Docker daemon is not available")
	}
	require.NotNil(t, c)

	// Since we are just testing the reaper, we'll try tracking some dummy IDs 
	// and observe if the client handles it properly without panicking.
	// In a full integration test we would create a container, track it,
	// cancel the context, and verify it's deleted. 
	
	ctx, cancel := context.WithCancel(context.Background())
	r := reaper.New(c)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		r.Watch(ctx)
	}()

	r.TrackContainer("dummy-container-id")
	r.TrackNetwork("dummy-network-id")
	r.TrackVolume("dummy-volume-id")

	// Trigger cleanup
	cancel()
	
	// Wait for cleanup to finish
	wg.Wait()
	
	// If we reach here without panic, the basic machinery works.
	// Ideally we assert that the calls were made, which is better done with a mock.
}
