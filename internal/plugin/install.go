package plugin

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// CopyPluginDir copies a plugin directory into pluginsDir/<name> and returns the destination path.
func CopyPluginDir(srcDir, pluginsDir string) (string, error) {
	manifestData, err := os.ReadFile(filepath.Join(srcDir, "manifest.json"))
	if err != nil {
		return "", fmt.Errorf("reading manifest.json: %w", err)
	}
	var meta struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(manifestData, &meta); err != nil {
		return "", fmt.Errorf("parsing manifest.json: %w", err)
	}
	destDir := filepath.Join(pluginsDir, meta.Name)
	if err := os.MkdirAll(destDir, 0o750); err != nil {
		return "", err
	}
	return destDir, copyDirContents(srcDir, destDir)
}

func copyDirContents(src, dest string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(destPath, 0o750)
		}
		return copyFile(path, destPath)
	})
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
