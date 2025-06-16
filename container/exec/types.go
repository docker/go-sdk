package exec

// ExecOptions is a struct that provides a default implementation for the Options method
// of the Executable interface.
type ExecOptions struct {
	opts []ProcessOption
}

func (ce ExecOptions) Options() []ProcessOption {
	return ce.opts
}

// RawCommand is a type that implements Executable and represents a command to be sent to a container
type RawCommand struct {
	ExecOptions
	cmds []string
}

func NewRawCommand(cmds []string, opts ...ProcessOption) RawCommand {
	return RawCommand{
		cmds: cmds,
		ExecOptions: ExecOptions{
			opts: opts,
		},
	}
}

// AsCommand returns the command as a slice of strings
func (r RawCommand) AsCommand() []string {
	return r.cmds
}
