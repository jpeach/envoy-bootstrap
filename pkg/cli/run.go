package cli

import (
	"github.com/spf13/cobra"
)

// NewRunCommand ...
func NewRunCommand() *cobra.Command {
	run := cobra.Command{
		Use:   "run [FLAGS ...]",
		Short: "Bootstrap and run Envoy",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	// TODO(jpeach): add flags.

	return Defaults(&run)
}
