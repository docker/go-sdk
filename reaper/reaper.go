package reaper

import (
	"context"
	"sync"
	"time"

	"github.com/moby/moby/client"

	sdkclient "github.com/docker/go-sdk/client"
)

// Reaper is responsible for tracking Docker resources and automatically removing
// them when its associated context is cancelled.
type Reaper struct {
	client sdkclient.SDKClient

	mu         sync.Mutex
	containers map[string]struct{}
	networks   map[string]struct{}
	volumes    map[string]struct{}
}

// New creates a new Ephemeral Resource Reaper attached to the given SDKClient.
func New(c sdkclient.SDKClient) *Reaper {
	return &Reaper{
		client:     c,
		containers: make(map[string]struct{}),
		networks:   make(map[string]struct{}),
		volumes:    make(map[string]struct{}),
	}
}

// Watch begins watching the context. When the context is done (e.g. cancelled
// or timed out), it will trigger the cleanup of all tracked resources.
// Watch blocks until the context is done and cleanup finishes.
func (r *Reaper) Watch(ctx context.Context) {
	<-ctx.Done()
	r.cleanUp()
}

// TrackContainer registers a container ID for deletion during cleanup.
func (r *Reaper) TrackContainer(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.containers[id] = struct{}{}
}

// TrackNetwork registers a network ID for deletion during cleanup.
func (r *Reaper) TrackNetwork(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.networks[id] = struct{}{}
}

// TrackVolume registers a volume ID for deletion during cleanup.
func (r *Reaper) TrackVolume(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.volumes[id] = struct{}{}
}

func (r *Reaper) cleanUp() {
	r.mu.Lock()
	containers := make([]string, 0, len(r.containers))
	for id := range r.containers {
		containers = append(containers, id)
	}
	networks := make([]string, 0, len(r.networks))
	for id := range r.networks {
		networks = append(networks, id)
	}
	volumes := make([]string, 0, len(r.volumes))
	for id := range r.volumes {
		volumes = append(volumes, id)
	}
	r.mu.Unlock()

	// Use a new background context with a timeout for deletion since the original context is already done.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	logger := r.client.Logger()

	// 1. Remove containers first
	var wg sync.WaitGroup
	for _, id := range containers {
		wg.Add(1)
		go func(containerID string) {
			defer wg.Done()
			if err := r.client.ContainerRemove(ctx, containerID, client.ContainerRemoveOptions{
				Force:         true,
				RemoveVolumes: true,
			}); err != nil {
				if logger != nil {
					logger.Warn("Failed to remove tracked container", "id", containerID, "error", err)
				}
			}
		}(id)
	}
	wg.Wait() // Wait for all containers to be removed before tackling networks/volumes.

	// 2. Remove networks and volumes in parallel after containers
	for _, id := range networks {
		wg.Add(1)
		go func(networkID string) {
			defer wg.Done()
			if err := r.client.NetworkRemove(ctx, networkID); err != nil {
				if logger != nil {
					logger.Warn("Failed to remove tracked network", "id", networkID, "error", err)
				}
			}
		}(id)
	}

	for _, id := range volumes {
		wg.Add(1)
		go func(volumeID string) {
			defer wg.Done()
			if err := r.client.VolumeRemove(ctx, volumeID, true); err != nil {
				if logger != nil {
					logger.Warn("Failed to remove tracked volume", "id", volumeID, "error", err)
				}
			}
		}(id)
	}

	wg.Wait()
}
