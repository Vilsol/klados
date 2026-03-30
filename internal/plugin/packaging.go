package plugin

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	mediaTypeConfig      = "application/vnd.klados.plugin.manifest.v1+json"
	mediaTypeWasm        = "application/vnd.klados.plugin.wasm.v1"
	mediaTypeUI          = "application/vnd.klados.plugin.ui.v1+tar+gzip"
	mediaTypeDescriptors = "application/vnd.klados.plugin.descriptors.v1+tar+gzip"
	ociLayoutVersion     = "1.0.0"
)

type ociLayout struct {
	ImageLayoutVersion string `json:"imageLayoutVersion"`
}

type ociDescriptor struct {
	MediaType string            `json:"mediaType"`
	Digest    string            `json:"digest"`
	Size      int64             `json:"size"`
	Title     string            `json:"title,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
}

type ociManifest struct {
	SchemaVersion int             `json:"schemaVersion"`
	MediaType     string          `json:"mediaType"`
	Config        ociDescriptor   `json:"config"`
	Layers        []ociDescriptor `json:"layers"`
}

type ociIndex struct {
	SchemaVersion int             `json:"schemaVersion"`
	MediaType     string          `json:"mediaType"`
	Manifests     []ociDescriptor `json:"manifests"`
}

// Pack creates an OCI Image Layout tar archive from a plugin directory.
// If compress is true, the output is gzip-compressed (.oci.tar.gz); otherwise .oci.tar.
// Returns the path to the created archive.
func Pack(pluginDir string, compress bool) (string, error) {
	manifestData, err := os.ReadFile(filepath.Join(pluginDir, "manifest.json"))
	if err != nil {
		return "", fmt.Errorf("reading manifest.json: %w", err)
	}

	var pluginMeta struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(manifestData, &pluginMeta); err != nil {
		return "", fmt.Errorf("parsing manifest.json: %w", err)
	}

	var fullManifest struct {
		Extensions *struct {
			Enrichers *struct {
				Wasm string `json:"wasm"`
			} `json:"enrichers,omitempty"`
		} `json:"extensions,omitempty"`
	}
	_ = json.Unmarshal(manifestData, &fullManifest)

	blobs := map[string][]byte{}
	configDesc := addBlob(blobs, manifestData, mediaTypeConfig)

	var layers []ociDescriptor

	if fullManifest.Extensions != nil && fullManifest.Extensions.Enrichers != nil {
		wasmRelPath := fullManifest.Extensions.Enrichers.Wasm
		wasmData, err := os.ReadFile(filepath.Join(pluginDir, wasmRelPath))
		if err != nil {
			return "", fmt.Errorf("reading wasm %q: %w", wasmRelPath, err)
		}
		wasmDesc := addBlob(blobs, wasmData, mediaTypeWasm)
		wasmDesc.Title = wasmRelPath
		layers = append(layers, wasmDesc)
	}

	if _, err := os.Stat(filepath.Join(pluginDir, "ui")); err == nil {
		uiTar, err := dirToTarGz(filepath.Join(pluginDir, "ui"), "ui")
		if err != nil {
			return "", fmt.Errorf("archiving ui dir: %w", err)
		}
		layers = append(layers, addBlob(blobs, uiTar, mediaTypeUI))
	}

	if _, err := os.Stat(filepath.Join(pluginDir, "descriptors")); err == nil {
		descTar, err := dirToTarGz(filepath.Join(pluginDir, "descriptors"), "descriptors")
		if err != nil {
			return "", fmt.Errorf("archiving descriptors dir: %w", err)
		}
		layers = append(layers, addBlob(blobs, descTar, mediaTypeDescriptors))
	}

	ociMan := ociManifest{
		SchemaVersion: 2,
		MediaType:     "application/vnd.oci.image.manifest.v1+json",
		Config:        configDesc,
		Layers:        layers,
	}
	ociManData, err := json.Marshal(ociMan)
	if err != nil {
		return "", fmt.Errorf("marshaling OCI manifest: %w", err)
	}
	ociManDesc := addBlob(blobs, ociManData, "application/vnd.oci.image.manifest.v1+json")
	ociManDesc.Labels = map[string]string{"org.opencontainers.image.title": pluginMeta.Name}

	index := ociIndex{
		SchemaVersion: 2,
		MediaType:     "application/vnd.oci.image.index.v1+json",
		Manifests:     []ociDescriptor{ociManDesc},
	}
	indexData, err := json.Marshal(index)
	if err != nil {
		return "", fmt.Errorf("marshaling OCI index: %w", err)
	}

	layoutData, _ := json.Marshal(ociLayout{ImageLayoutVersion: ociLayoutVersion})

	ext := ".oci.tar"
	if compress {
		ext = ".oci.tar.gz"
	}
	outPath := filepath.Join(pluginDir, pluginMeta.Name+"-"+pluginMeta.Version+ext)

	f, err := os.Create(outPath)
	if err != nil {
		return "", fmt.Errorf("creating archive: %w", err)
	}
	defer f.Close()

	var tw *tar.Writer
	if compress {
		gw := gzip.NewWriter(f)
		defer gw.Close()
		tw = tar.NewWriter(gw)
	} else {
		tw = tar.NewWriter(f)
	}
	defer tw.Close()

	for _, entry := range []struct {
		name string
		data []byte
	}{
		{"oci-layout", layoutData},
		{"index.json", indexData},
	} {
		if err := writeTarEntry(tw, entry.name, entry.data); err != nil {
			return "", err
		}
	}

	for digest, data := range blobs {
		path := "blobs/sha256/" + strings.TrimPrefix(digest, "sha256:")
		if err := writeTarEntry(tw, path, data); err != nil {
			return "", err
		}
	}

	return outPath, nil
}

// Unpack extracts a plugin from an OCI tar archive into pluginsDir.
// Detects gzip compression automatically via magic bytes (1f 8b).
func Unpack(archivePath, pluginsDir string) error {
	entries, err := readArchiveEntries(archivePath)
	if err != nil {
		return err
	}

	indexData, ok := entries["index.json"]
	if !ok {
		return fmt.Errorf("index.json not found in archive")
	}
	var index ociIndex
	if err := json.Unmarshal(indexData, &index); err != nil {
		return fmt.Errorf("parsing index.json: %w", err)
	}
	if len(index.Manifests) == 0 {
		return fmt.Errorf("no manifests in index.json")
	}

	ociManData := blobData(entries, index.Manifests[0].Digest)
	if ociManData == nil {
		return fmt.Errorf("manifest blob not found: %s", index.Manifests[0].Digest)
	}
	var ociMan ociManifest
	if err := json.Unmarshal(ociManData, &ociMan); err != nil {
		return fmt.Errorf("parsing OCI manifest: %w", err)
	}

	return extractPlugin(ociMan, func(d string) []byte { return blobData(entries, d) }, pluginsDir)
}

// readArchiveEntries reads a .oci.tar or .oci.tar.gz into a name→bytes map.
func readArchiveEntries(archivePath string) (map[string][]byte, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return nil, fmt.Errorf("opening archive: %w", err)
	}
	defer f.Close()

	magic := make([]byte, 2)
	if _, err := io.ReadFull(f, magic); err != nil {
		return nil, fmt.Errorf("reading magic bytes: %w", err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seeking: %w", err)
	}

	var tr *tar.Reader
	if magic[0] == 0x1f && magic[1] == 0x8b {
		gr, err := gzip.NewReader(f)
		if err != nil {
			return nil, fmt.Errorf("creating gzip reader: %w", err)
		}
		defer gr.Close()
		tr = tar.NewReader(gr)
	} else {
		tr = tar.NewReader(f)
	}

	entries := map[string][]byte{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading tar entry: %w", err)
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			return nil, fmt.Errorf("reading tar data for %s: %w", hdr.Name, err)
		}
		entries[hdr.Name] = data
	}
	return entries, nil
}

// extractPlugin writes a plugin from a parsed manifest + blob-lookup function into pluginsDir.
func extractPlugin(manifest ociManifest, blobs func(digest string) []byte, pluginsDir string) error {
	configData := blobs(manifest.Config.Digest)
	if configData == nil {
		return fmt.Errorf("config blob not found: %s", manifest.Config.Digest)
	}
	var pluginMeta struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(configData, &pluginMeta); err != nil {
		return fmt.Errorf("parsing plugin manifest: %w", err)
	}
	if pluginMeta.Name == "" {
		return fmt.Errorf("plugin name empty in manifest")
	}

	pluginDir := filepath.Join(pluginsDir, pluginMeta.Name)
	if err := os.MkdirAll(pluginDir, 0o750); err != nil {
		return fmt.Errorf("creating plugin dir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), configData, 0o640); err != nil {
		return fmt.Errorf("writing manifest.json: %w", err)
	}

	for _, layer := range manifest.Layers {
		data := blobs(layer.Digest)
		if data == nil {
			return fmt.Errorf("layer blob not found: %s", layer.Digest)
		}
		switch layer.MediaType {
		case mediaTypeWasm:
			wasmPath := layer.Title
			if wasmPath == "" {
				wasmPath = "plugin.wasm"
			}
			dest := filepath.Join(pluginDir, wasmPath)
			if err := os.MkdirAll(filepath.Dir(dest), 0o750); err != nil {
				return fmt.Errorf("creating wasm parent dir: %w", err)
			}
			if err := os.WriteFile(dest, data, 0o640); err != nil {
				return fmt.Errorf("writing wasm: %w", err)
			}
		case mediaTypeUI, mediaTypeDescriptors:
			if err := extractTarGz(data, pluginDir); err != nil {
				return fmt.Errorf("extracting layer %s: %w", layer.MediaType, err)
			}
		}
	}

	return nil
}

func addBlob(blobs map[string][]byte, data []byte, mediaType string) ociDescriptor {
	h := sha256.Sum256(data)
	digest := "sha256:" + hex.EncodeToString(h[:])
	blobs[digest] = data
	return ociDescriptor{
		MediaType: mediaType,
		Digest:    digest,
		Size:      int64(len(data)),
	}
}

func blobData(entries map[string][]byte, digest string) []byte {
	return entries["blobs/sha256/"+strings.TrimPrefix(digest, "sha256:")]
}

// dirToTarGz creates an in-memory tar+gzip of a directory.
// All paths inside are prefixed with tarPrefix (e.g. "ui").
func dirToTarGz(dir, tarPrefix string) ([]byte, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return writeTarEntry(tw, tarPrefix+"/"+filepath.ToSlash(rel), data)
	})
	if err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeTarEntry(tw *tar.Writer, name string, data []byte) error {
	if err := tw.WriteHeader(&tar.Header{
		Name: name,
		Mode: 0o640,
		Size: int64(len(data)),
	}); err != nil {
		return fmt.Errorf("writing tar header for %s: %w", name, err)
	}
	if _, err := tw.Write(data); err != nil {
		return fmt.Errorf("writing tar data for %s: %w", name, err)
	}
	return nil
}

func extractTarGz(data []byte, destDir string) error {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("creating gzip reader: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar: %w", err)
		}
		clean := filepath.Clean(hdr.Name)
		if strings.HasPrefix(clean, "..") {
			continue
		}
		dest := filepath.Join(destDir, clean)
		if hdr.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(dest, 0o750); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o750); err != nil {
			return err
		}
		content, err := io.ReadAll(tr)
		if err != nil {
			return err
		}
		if err := os.WriteFile(dest, content, 0o640); err != nil {
			return err
		}
	}
	return nil
}
