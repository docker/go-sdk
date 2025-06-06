package dockercontainer

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-sdk/dockerclient"
	cexec "github.com/docker/go-sdk/dockercontainer/exec"
	"github.com/docker/go-sdk/dockercontainer/wait"
)

// Container represents a container
type Container struct {
	dockerClient *dockerclient.Client

	// ID the Container ID
	ID string

	// shortID the short Container ID, using the first 12 characters of the ID
	shortID string

	// WaitingFor the waiting strategy to use for the container.
	WaitingFor wait.Strategy

	// TODO: Remove locking and wait group once the deprecated StartLogProducer and
	// StopLogProducer have been removed and hence logging can only be started and
	// stopped once.

	// logProductionCancel is used to signal the log production to stop.
	logProductionCancel context.CancelCauseFunc
	logProductionCtx    context.Context

	logProductionTimeout *time.Duration

	// Image the image to use for the container.
	Image string

	// exposedPorts the ports exposed by the container.
	exposedPorts []string

	// logger the logger to use for the container.
	logger *slog.Logger

	// lifecycleHooks the lifecycle hooks to use for the container.
	lifecycleHooks []LifecycleHooks

	// consumers the log consumers to use for the container.
	consumers []LogConsumer

	// isRunning the flag to check if the container is running.
	isRunning bool
}

// Exec executes a command in the current container.
// It returns the exit status of the executed command, an [io.Reader] containing the combined
// stdout and stderr, and any encountered error. Note that reading directly from the [io.Reader]
// may result in unexpected bytes due to custom stream multiplexing headers.
// Use [cexec.Multiplexed] option to read the combined output without the multiplexing headers.
// Alternatively, to separate the stdout and stderr from [io.Reader] and interpret these headers properly,
// [github.com/docker/docker/pkg/stdcopy.StdCopy] from the Docker API should be used.
func (c *Container) Exec(ctx context.Context, cmd []string, options ...cexec.ProcessOption) (int, io.Reader, error) {
	processOptions := cexec.NewProcessOptions(cmd)

	// processing all the options in a first loop because for the multiplexed option
	// we first need to have a containerExecCreateResponse
	for _, o := range options {
		o.Apply(processOptions)
	}

	response, err := c.dockerClient.Client().ContainerExecCreate(ctx, c.ID, processOptions.ExecConfig)
	if err != nil {
		return 0, nil, fmt.Errorf("container exec create: %w", err)
	}

	hijack, err := c.dockerClient.Client().ContainerExecAttach(ctx, response.ID, container.ExecAttachOptions{})
	if err != nil {
		return 0, nil, fmt.Errorf("container exec attach: %w", err)
	}

	processOptions.Reader = hijack.Reader

	// second loop to process the multiplexed option, as now we have a reader
	// from the created exec response.
	for _, o := range options {
		o.Apply(processOptions)
	}

	var exitCode int
	for {
		execResp, err := c.dockerClient.Client().ContainerExecInspect(ctx, response.ID)
		if err != nil {
			return 0, nil, fmt.Errorf("container exec inspect: %w", err)
		}

		if !execResp.Running {
			exitCode = execResp.ExitCode
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	return exitCode, processOptions.Reader, nil
}

// Host gets host (ip or name) of the docker daemon where the container port is exposed
// Warning: this is based on your Docker host setting. Will fail if using an SSH tunnel
func (c *Container) Host(ctx context.Context) (string, error) {
	host, err := c.dockerClient.DaemonHost(ctx)
	if err != nil {
		return "", err
	}
	return host, nil
}
