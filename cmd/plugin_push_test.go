package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func newTestPluginCmd() *cobra.Command {
	root := &cobra.Command{Use: "klados", SilenceUsage: true, SilenceErrors: true}
	root.AddCommand(pluginCmd)
	return root
}

func TestPluginPush_NoArgs(t *testing.T) {
	root := newTestPluginCmd()
	root.SetArgs([]string{"plugin", "push"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error with no args, got nil")
	}
}

func TestPluginPush_WrongExtension(t *testing.T) {
	root := newTestPluginCmd()
	root.SetArgs([]string{"plugin", "push", "archive.zip", "oci://ghcr.io/owner/plugin:v1"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for wrong extension, got nil")
	}
	if err.Error() == "" {
		t.Fatal("expected non-empty error message")
	}
	// Should mention extension
	msg := err.Error()
	if msg != "archive must end in .oci.tar.gz or .oci.tar" {
		t.Fatalf("unexpected error: %s", msg)
	}
}

func TestPluginPush_NonExistentArchive(t *testing.T) {
	root := newTestPluginCmd()
	root.SetArgs([]string{"plugin", "push", "/nonexistent/path/plugin.oci.tar.gz", "oci://ghcr.io/owner/plugin:v1"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error for non-existent archive, got nil")
	}
}
