package cmd

import (
	"fmt"

	"github.com/Vilsol/klados/internal/plugin"
	"github.com/spf13/cobra"
)

func init() {
	var noCompress bool

	packCmd := &cobra.Command{
		Use:   "pack <dir>",
		Short: "Pack a plugin directory into an OCI archive",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out, err := plugin.Pack(args[0], !noCompress)
			if err != nil {
				return fmt.Errorf("pack failed: %w", err)
			}
			fmt.Println(out)
			return nil
		},
	}

	packCmd.Flags().BoolVar(&noCompress, "no-compress", false, "produce .oci.tar instead of .oci.tar.gz")
	pluginCmd.AddCommand(packCmd)
}
