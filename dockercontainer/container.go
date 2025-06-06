package dockercontainer

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-sdk/dockerclient"
	cexec "github.com/docker/go-sdk/dockercontainer/exec"
	"github.com/docker/go-sdk/dockercontainer/wait"
)

// Container represents a container
type Container struct {
	// ID the Container ID
	ID string

	// WaitingFor the waiting strategy to use for the container.
	WaitingFor wait.Strategy

	// Image the image to use for the container.
	Image string
}

// Exec executes a command in the current container.
// It returns the exit status of the executed command, an [io.Reader] containing the combined
// stdout and stderr, and any encountered error. Note that reading directly from the [io.Reader]
// may result in unexpected bytes due to custom stream multiplexing headers.
// Use [cexec.Multiplexed] option to read the combined output without the multiplexing headers.
// Alternatively, to separate the stdout and stderr from [io.Reader] and interpret these headers properly,
// [github.com/docker/docker/pkg/stdcopy.StdCopy] from the Docker API should be used.
func (c *Container) Exec(ctx context.Context, cmd []string, options ...cexec.ProcessOption) (int, io.Reader, error) {
	cli, err := dockerclient.NewClient(ctx)
	if err != nil {
		return 0, nil, fmt.Errorf("new client: %w", err)
	}

	processOptions := cexec.NewProcessOptions(cmd)

	// processing all the options in a first loop because for the multiplexed option
	// we first need to have a containerExecCreateResponse
	for _, o := range options {
		o.Apply(processOptions)
	}

	response, err := cli.Client().ContainerExecCreate(ctx, c.ID, processOptions.ExecConfig)
	if err != nil {
		return 0, nil, fmt.Errorf("container exec create: %w", err)
	}

	hijack, err := cli.Client().ContainerExecAttach(ctx, response.ID, container.ExecAttachOptions{})
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
		execResp, err := cli.Client().ContainerExecInspect(ctx, response.ID)
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
