package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Vilsol/klados/internal/plugin"
	"github.com/spf13/cobra"
)

func init() {
	var username, password, token string
	var insecure bool

	pushCmd := &cobra.Command{
		Use:   "push <archive> <ref>",
		Short: "Push a plugin archive to an OCI registry",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			archive, ref := args[0], args[1]
			if !strings.HasSuffix(archive, ".oci.tar.gz") && !strings.HasSuffix(archive, ".oci.tar") {
				return fmt.Errorf("archive must end in .oci.tar.gz or .oci.tar")
			}
			if _, err := os.Stat(archive); err != nil {
				return fmt.Errorf("archive not found: %w", err)
			}
			opts := plugin.RemoteOpts{Username: username, Password: password, Token: token, Insecure: insecure}
			if err := plugin.PushToRegistry(archive, ref, opts); err != nil {
				if errors.Is(err, plugin.ErrAuthRequired) {
					return fmt.Errorf("authentication required — use --username/--password or --token")
				}
				return err
			}
			fmt.Printf("pushed %s\n", ref)
			return nil
		},
	}

	pushCmd.Flags().StringVarP(&username, "username", "u", "", "registry username")
	pushCmd.Flags().StringVarP(&password, "password", "p", "", "registry password")
	pushCmd.Flags().StringVarP(&token, "token", "t", "", "registry bearer token")
	pushCmd.Flags().BoolVar(&insecure, "insecure", false, "use plain HTTP")
	pluginCmd.AddCommand(pushCmd)
}
