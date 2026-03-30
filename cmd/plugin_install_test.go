package cmd

import (
	"os"
	"strings"
	"testing"
)

func TestPluginInstall_NoArgs(t *testing.T) {
	root := newTestPluginCmd()
	root.SetArgs([]string{"plugin", "install"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error with no args, got nil")
	}
}

func TestPluginInstall_NonExistentPath(t *testing.T) {
	root := newTestPluginCmd()
	root.SetArgs([]string{"plugin", "install", "/nonexistent/path/that/does/not/exist"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for non-existent path, got nil")
	}
}

func TestPluginInstall_UnrecognisedFormat(t *testing.T) {
	// Create a real file with an unrecognised extension so stat succeeds but format check fails.
	f, err := os.CreateTemp("", "plugin-test-*.zip")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })

	root := newTestPluginCmd()
	root.SetArgs([]string{"plugin", "install", f.Name()})
	err = root.Execute()
	if err == nil {
		t.Fatal("expected error for unrecognised format, got nil")
	}
	if !strings.Contains(err.Error(), "unrecognised format") {
		t.Fatalf("expected 'unrecognised format' in error, got: %s", err.Error())
	}
}
