package plugin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
)

// RemoteOpts configures registry authentication and transport.
type RemoteOpts struct {
	Username, Password, Token string
	Insecure                  bool
}

// ErrAuthRequired is returned when registry credentials are missing or invalid.
var ErrAuthRequired = errors.New("authentication required")

func dockerConfigPath() string {
	return filepath.Join(os.Getenv("HOME"), ".docker", "config.json")
}

func resolveCredential(opts RemoteOpts) auth.CredentialFunc {
	if opts.Token != "" {
		return func(_ context.Context, _ string) (auth.Credential, error) {
			return auth.Credential{AccessToken: opts.Token}, nil
		}
	}
	if opts.Username != "" && opts.Password != "" {
		return func(_ context.Context, _ string) (auth.Credential, error) {
			return auth.Credential{Username: opts.Username, Password: opts.Password}, nil
		}
	}
	path := dockerConfigPath()
	if _, err := os.Stat(path); err == nil {
		if store, err := credentials.NewFileStore(path); err == nil {
			return credentials.Credential(store)
		}
	}
	return func(_ context.Context, _ string) (auth.Credential, error) {
		return auth.EmptyCredential, nil
	}
}

func newRepo(ref string, opts RemoteOpts) (*remote.Repository, error) {
	ref = strings.TrimPrefix(ref, "oci://")
	repo, err := remote.NewRepository(ref)
	if err != nil {
		return nil, fmt.Errorf("parsing reference %q: %w", ref, err)
	}
	repo.PlainHTTP = opts.Insecure
	repo.Client = &auth.Client{Credential: resolveCredential(opts)}
	return repo, nil
}

func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unauthorized") ||
		strings.Contains(msg, "authentication required") ||
		strings.Contains(msg, "credential not found") ||
		strings.Contains(msg, "credentials not found") ||
		strings.Contains(msg, "status 401") ||
		strings.Contains(msg, "status 403")
}

func unpackDir(layoutDir, pluginsDir string) error {
	indexData, err := os.ReadFile(filepath.Join(layoutDir, "index.json"))
	if err != nil {
		return fmt.Errorf("reading index.json: %w", err)
	}
	var index ociIndex
	if err := json.Unmarshal(indexData, &index); err != nil {
		return fmt.Errorf("parsing index.json: %w", err)
	}
	if len(index.Manifests) == 0 {
		return fmt.Errorf("no manifests in index.json")
	}

	blobsDir := filepath.Join(layoutDir, "blobs", "sha256")
	readBlob := func(digest string) []byte {
		data, _ := os.ReadFile(filepath.Join(blobsDir, strings.TrimPrefix(digest, "sha256:")))
		return data
	}

	manData := readBlob(index.Manifests[0].Digest)
	if manData == nil {
		return fmt.Errorf("manifest blob not found: %s", index.Manifests[0].Digest)
	}
	var manifest ociManifest
	if err := json.Unmarshal(manData, &manifest); err != nil {
		return fmt.Errorf("parsing OCI manifest: %w", err)
	}

	return extractPlugin(manifest, readBlob, pluginsDir)
}

// PullFromRegistry pulls a plugin from a registry into pluginsDir.
func PullFromRegistry(ref, pluginsDir string, opts RemoteOpts) error {
	tempDir, err := os.MkdirTemp("", "klados-pull-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := oci.New(tempDir)
	if err != nil {
		return fmt.Errorf("creating layout store: %w", err)
	}

	repo, err := newRepo(ref, opts)
	if err != nil {
		return err
	}

	tag := repo.Reference.Reference
	if tag == "" {
		tag = "latest"
	}

	ctx := context.Background()
	if _, err := oras.Copy(ctx, repo, tag, store, tag, oras.DefaultCopyOptions); err != nil {
		if isAuthError(err) {
			return ErrAuthRequired
		}
		return fmt.Errorf("pulling %s: %w", ref, err)
	}

	return unpackDir(tempDir, pluginsDir)
}

// PushToRegistry pushes a plugin archive to a registry.
func PushToRegistry(archivePath, ref string, opts RemoteOpts) error {
	entries, err := readArchiveEntries(archivePath)
	if err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp("", "klados-push-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	for name, data := range entries {
		dest := filepath.Join(tempDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(dest), 0o750); err != nil {
			return fmt.Errorf("creating parent for %s: %w", name, err)
		}
		if err := os.WriteFile(dest, data, 0o640); err != nil {
			return fmt.Errorf("writing %s: %w", name, err)
		}
	}

	store, err := oci.New(tempDir)
	if err != nil {
		return fmt.Errorf("creating layout store: %w", err)
	}

	indexData := entries["index.json"]
	if indexData == nil {
		return fmt.Errorf("index.json not found in archive")
	}
	var index ociIndex
	if err := json.Unmarshal(indexData, &index); err != nil {
		return fmt.Errorf("parsing index.json: %w", err)
	}
	if len(index.Manifests) == 0 {
		return fmt.Errorf("no manifests in index.json")
	}

	repo, err := newRepo(ref, opts)
	if err != nil {
		return err
	}

	tag := repo.Reference.Reference
	if tag == "" {
		tag = "latest"
	}

	ctx := context.Background()
	manifestDigest := index.Manifests[0].Digest
	if _, err := oras.Copy(ctx, store, manifestDigest, repo, tag, oras.DefaultCopyOptions); err != nil {
		if isAuthError(err) {
			return ErrAuthRequired
		}
		return fmt.Errorf("pushing %s: %w", ref, err)
	}

	return nil
}

// SaveDockerCredentials writes credentials for host into ~/.docker/config.json.
func SaveDockerCredentials(host, username, password string) error {
	configPath := dockerConfigPath()

	var cfg map[string]interface{}
	data, err := os.ReadFile(configPath)
	if err == nil {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return fmt.Errorf("parsing docker config: %w", err)
		}
	} else if os.IsNotExist(err) {
		cfg = map[string]interface{}{}
	} else {
		return fmt.Errorf("reading docker config: %w", err)
	}

	auths, _ := cfg["auths"].(map[string]interface{})
	if auths == nil {
		auths = map[string]interface{}{}
		cfg["auths"] = auths
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	auths[host] = map[string]interface{}{"auth": encoded}

	out, err := json.MarshalIndent(cfg, "", "\t")
	if err != nil {
		return fmt.Errorf("marshaling docker config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		return fmt.Errorf("creating .docker dir: %w", err)
	}
	return os.WriteFile(configPath, out, 0o600)
}
