package drift

import (
	"github.com/spf13/cobra"
)

// NewCmdDrift returns the drift parent command.
func NewCmdDrift() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "drift",
		Short: "Check infrastructure drift detection status",
	}

	cmd.AddCommand(newCmdDriftList())
	cmd.AddCommand(newCmdDriftShow())

	return cmd
}
