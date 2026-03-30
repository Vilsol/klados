package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Vilsol/klados/internal/plugin"
	"github.com/adrg/xdg"
	"github.com/spf13/cobra"
)

func init() {
	var username, password, token string
	var insecure bool

	installCmd := &cobra.Command{
		Use:   "install <path-or-oci://ref>",
		Short: "Install a plugin from a local path or OCI registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			arg := args[0]
			pluginsDir := filepath.Join(xdg.DataHome, "klados", "plugins")
			opts := plugin.RemoteOpts{Username: username, Password: password, Token: token, Insecure: insecure}

			var destDir string

			switch {
			case strings.HasPrefix(arg, "oci://"):
				if err := plugin.PullFromRegistry(arg, pluginsDir, opts); err != nil {
					if errors.Is(err, plugin.ErrAuthRequired) {
						return fmt.Errorf("authentication required — use --username/--password or --token")
					}
					return err
				}
				destDir = pluginsDir

			default:
				info, err := os.Stat(arg)
				if err != nil {
					return fmt.Errorf("accessing path: %w", err)
				}
				if info.IsDir() {
					destDir, err = plugin.CopyPluginDir(arg, pluginsDir)
					if err != nil {
						return fmt.Errorf("copying plugin dir: %w", err)
					}
				} else if strings.HasSuffix(arg, ".oci.tar.gz") || strings.HasSuffix(arg, ".oci.tar") {
					if err := plugin.Unpack(arg, pluginsDir); err != nil {
						return fmt.Errorf("unpacking plugin: %w", err)
					}
					destDir = pluginsDir
				} else {
					return fmt.Errorf("unrecognised format: argument must be oci:// ref, a directory, or a .oci.tar/.oci.tar.gz file")
				}
			}

			fmt.Printf("installed plugin to %s\n", destDir)
			return nil
		},
	}

	installCmd.Flags().StringVarP(&username, "username", "u", "", "registry username")
	installCmd.Flags().StringVarP(&password, "password", "p", "", "registry password")
	installCmd.Flags().StringVarP(&token, "token", "t", "", "registry bearer token")
	installCmd.Flags().BoolVar(&insecure, "insecure", false, "use plain HTTP")
	pluginCmd.AddCommand(installCmd)
}
