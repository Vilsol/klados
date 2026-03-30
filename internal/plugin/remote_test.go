package plugin

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/MarvinJWendt/testza"
)

// mockReg is a minimal OCI distribution spec server for testing.
type mockReg struct {
	mu         sync.Mutex
	blobs      map[string][]byte // sha256 hex (no prefix) → data
	manifests  map[string][]byte // "repo/ref" or "repo@digest" → data
	mediaTypes map[string]string // same keys as manifests → content-type

	// auth mode: "none", "basic", "bearer", "reject"
	mode  string
	user  string
	pass  string
	token string

	server *httptest.Server
}

func newMockReg(t *testing.T) *mockReg {
	t.Helper()
	r := &mockReg{
		blobs:      make(map[string][]byte),
		manifests:  make(map[string][]byte),
		mediaTypes: make(map[string]string),
		mode:       "none",
	}
	r.server = httptest.NewServer(r)
	t.Cleanup(r.server.Close)
	return r
}

// ociRef returns an oci:// reference suitable for PullFromRegistry/PushToRegistry.
func (r *mockReg) ociRef(repo, tag string) string {
	host := strings.TrimPrefix(r.server.URL, "http://")
	return "oci://" + host + "/" + repo + ":" + tag
}

func (r *mockReg) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.checkAuth(req) {
		r.sendUnauthorized(w)
		return
	}

	path := req.URL.Path

	if path == "/v2/" {
		w.WriteHeader(http.StatusOK)
		return
	}

	rest := strings.TrimPrefix(path, "/v2/")

	switch {
	// PUT blob upload complete: /v2/{repo}/blobs/uploads/{id}?digest=...
	case strings.Contains(rest, "/blobs/uploads/") && req.Method == http.MethodPut:
		digest := req.URL.Query().Get("digest")
		hash := strings.TrimPrefix(digest, "sha256:")
		body, _ := io.ReadAll(req.Body)
		r.blobs[hash] = body
		w.Header().Set("Docker-Content-Digest", digest)
		w.WriteHeader(http.StatusCreated)

	// POST blob upload initiation: /v2/{repo}/blobs/uploads/
	case strings.Contains(rest, "/blobs/uploads") && req.Method == http.MethodPost:
		idx := strings.LastIndex(rest, "/blobs/uploads")
		repo := rest[:idx]
		w.Header().Set("Location", "/v2/"+repo+"/blobs/uploads/upload-1")
		w.WriteHeader(http.StatusAccepted)

	// HEAD/GET blob: /v2/{repo}/blobs/{digest}
	case strings.Contains(rest, "/blobs/") && !strings.Contains(rest, "/uploads"):
		idx := strings.LastIndex(rest, "/blobs/")
		digest := rest[idx+len("/blobs/"):]
		hash := strings.TrimPrefix(digest, "sha256:")
		data, ok := r.blobs[hash]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Docker-Content-Digest", "sha256:"+hash)
		if req.Method == http.MethodHead {
			w.Header().Set("Content-Length", strconv.Itoa(len(data)))
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(data) //nolint:errcheck

	// PUT/GET/HEAD manifest: /v2/{repo}/manifests/{ref}
	case strings.Contains(rest, "/manifests/"):
		idx := strings.LastIndex(rest, "/manifests/")
		repo := rest[:idx]
		ref := rest[idx+len("/manifests/"):]

		switch req.Method {
		case http.MethodPut:
			body, _ := io.ReadAll(req.Body)
			ct := req.Header.Get("Content-Type")
			key := repo + "/" + ref
			r.manifests[key] = body
			r.mediaTypes[key] = ct
			h := sha256.Sum256(body)
			dgst := "sha256:" + hex.EncodeToString(h[:])
			// also index by digest
			dgstKey := repo + "@" + dgst
			r.manifests[dgstKey] = body
			r.mediaTypes[dgstKey] = ct
			w.Header().Set("Docker-Content-Digest", dgst)
			w.WriteHeader(http.StatusCreated)

		case http.MethodGet, http.MethodHead:
			key := repo + "/" + ref
			data, ok := r.manifests[key]
			ct := r.mediaTypes[key]
			if !ok {
				key = repo + "@" + ref
				data, ok = r.manifests[key]
				ct = r.mediaTypes[key]
			}
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			if ct == "" {
				ct = "application/vnd.oci.image.manifest.v1+json"
			}
			h := sha256.Sum256(data)
			dgst := "sha256:" + hex.EncodeToString(h[:])
			w.Header().Set("Content-Type", ct)
			w.Header().Set("Docker-Content-Digest", dgst)
			if req.Method == http.MethodHead {
				w.Header().Set("Content-Length", strconv.Itoa(len(data)))
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write(data) //nolint:errcheck

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}

	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (r *mockReg) checkAuth(req *http.Request) bool {
	switch r.mode {
	case "none", "":
		return true
	case "reject":
		return false
	case "basic":
		u, p, ok := req.BasicAuth()
		return ok && u == r.user && p == r.pass
	case "bearer":
		return req.Header.Get("Authorization") == "Bearer "+r.token
	}
	return true
}

func (r *mockReg) sendUnauthorized(w http.ResponseWriter) {
	switch r.mode {
	case "bearer":
		realm := r.server.URL + "/token"
		w.Header().Set("WWW-Authenticate", `Bearer realm="`+realm+`",service="test"`)
	default:
		w.Header().Set("WWW-Authenticate", `Basic realm="test"`)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"errors":[{"code":"UNAUTHORIZED","message":"authentication required"}]}`)) //nolint:errcheck
}

// makeTestArchive creates a minimal plugin and returns the path to its .oci.tar.gz.
func makeTestArchive(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	manifest := []byte(`{"name":"test-plugin","version":"0.1.0"}`)
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), manifest, 0o640); err != nil {
		t.Fatal(err)
	}
	archivePath, err := Pack(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	return archivePath
}

func TestRemoteAnonymousPull(t *testing.T) {
	archive := makeTestArchive(t)
	reg := newMockReg(t)
	ref := reg.ociRef("test-plugin", "v0.1.0")
	opts := RemoteOpts{Insecure: true}

	testza.AssertNoError(t, PushToRegistry(archive, ref, opts))

	pluginsDir := t.TempDir()
	testza.AssertNoError(t, PullFromRegistry(ref, pluginsDir, opts))

	data, err := os.ReadFile(filepath.Join(pluginsDir, "test-plugin", "manifest.json"))
	testza.AssertNoError(t, err)
	var m struct {
		Name string `json:"name"`
	}
	testza.AssertNoError(t, json.Unmarshal(data, &m))
	testza.AssertEqual(t, "test-plugin", m.Name)
}

func TestRemoteBasicAuth(t *testing.T) {
	archive := makeTestArchive(t)
	reg := newMockReg(t)
	reg.mode = "basic"
	reg.user = "alice"
	reg.pass = "secret"

	opts := RemoteOpts{Username: "alice", Password: "secret", Insecure: true}
	ref := reg.ociRef("test-plugin", "v0.1.0")

	testza.AssertNoError(t, PushToRegistry(archive, ref, opts))

	pluginsDir := t.TempDir()
	testza.AssertNoError(t, PullFromRegistry(ref, pluginsDir, opts))
	testza.AssertTrue(t, fileExistsRemote(filepath.Join(pluginsDir, "test-plugin", "manifest.json")))
}

func TestRemoteTokenAuth(t *testing.T) {
	archive := makeTestArchive(t)
	reg := newMockReg(t)
	reg.mode = "bearer"
	reg.token = "my-secret-token"

	opts := RemoteOpts{Token: "my-secret-token", Insecure: true}
	ref := reg.ociRef("test-plugin", "v0.1.0")

	testza.AssertNoError(t, PushToRegistry(archive, ref, opts))

	pluginsDir := t.TempDir()
	testza.AssertNoError(t, PullFromRegistry(ref, pluginsDir, opts))
	testza.AssertTrue(t, fileExistsRemote(filepath.Join(pluginsDir, "test-plugin", "manifest.json")))
}

func TestRemoteDockerConfigPickup(t *testing.T) {
	archive := makeTestArchive(t)
	reg := newMockReg(t)
	reg.mode = "basic"
	reg.user = "docker-user"
	reg.pass = "docker-pass"

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	host := strings.TrimPrefix(reg.server.URL, "http://")
	if err := SaveDockerCredentials(host, "docker-user", "docker-pass"); err != nil {
		t.Fatal(err)
	}

	ref := reg.ociRef("test-plugin", "v0.1.0")

	// Push with explicit creds to populate registry
	pushOpts := RemoteOpts{Username: "docker-user", Password: "docker-pass", Insecure: true}
	testza.AssertNoError(t, PushToRegistry(archive, ref, pushOpts))

	// Pull with no explicit creds — relies on docker config
	pluginsDir := t.TempDir()
	pullOpts := RemoteOpts{Insecure: true}
	testza.AssertNoError(t, PullFromRegistry(ref, pluginsDir, pullOpts))
	testza.AssertTrue(t, fileExistsRemote(filepath.Join(pluginsDir, "test-plugin", "manifest.json")))
}

func TestRemoteAuthError(t *testing.T) {
	archive := makeTestArchive(t)
	reg := newMockReg(t)
	reg.mode = "reject"

	ref := reg.ociRef("test-plugin", "v0.1.0")
	opts := RemoteOpts{Insecure: true}

	err := PushToRegistry(archive, ref, opts)
	testza.AssertTrue(t, errors.Is(err, ErrAuthRequired))

	err = PullFromRegistry(ref, t.TempDir(), opts)
	testza.AssertTrue(t, errors.Is(err, ErrAuthRequired))
}

func TestRemotePushPullRoundTrip(t *testing.T) {
	archive := makeTestArchive(t)
	reg := newMockReg(t)
	ref := reg.ociRef("test-plugin", "v0.1.0")
	opts := RemoteOpts{Insecure: true}

	testza.AssertNoError(t, PushToRegistry(archive, ref, opts))

	pluginsDir := t.TempDir()
	testza.AssertNoError(t, PullFromRegistry(ref, pluginsDir, opts))

	// Compare pulled manifest.json against unpacking the original archive
	origPluginsDir := t.TempDir()
	if err := Unpack(archive, origPluginsDir); err != nil {
		t.Fatal(err)
	}

	orig, err := os.ReadFile(filepath.Join(origPluginsDir, "test-plugin", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	pulled, err := os.ReadFile(filepath.Join(pluginsDir, "test-plugin", "manifest.json"))
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, string(orig), string(pulled))
}

func TestRemoteInsecure(t *testing.T) {
	archive := makeTestArchive(t)
	reg := newMockReg(t)
	ref := reg.ociRef("test-plugin", "v0.1.0")

	// Insecure: true allows plain-HTTP connections
	opts := RemoteOpts{Insecure: true}
	testza.AssertNoError(t, PushToRegistry(archive, ref, opts))

	pluginsDir := t.TempDir()
	testza.AssertNoError(t, PullFromRegistry(ref, pluginsDir, opts))
	testza.AssertTrue(t, fileExistsRemote(filepath.Join(pluginsDir, "test-plugin", "manifest.json")))
}

func TestSaveDockerCredentials(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	configPath := filepath.Join(tmpHome, ".docker", "config.json")

	// Creates file when absent
	testza.AssertNoError(t, SaveDockerCredentials("registry1.example.com", "user1", "pass1"))

	data, err := os.ReadFile(configPath)
	testza.AssertNoError(t, err)

	var cfg map[string]interface{}
	testza.AssertNoError(t, json.Unmarshal(data, &cfg))

	auths := cfg["auths"].(map[string]interface{})
	reg1 := auths["registry1.example.com"].(map[string]interface{})
	decoded1, err := base64.StdEncoding.DecodeString(reg1["auth"].(string))
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "user1:pass1", string(decoded1))

	// Merges without clobbering other hosts
	testza.AssertNoError(t, SaveDockerCredentials("registry2.example.com", "user2", "pass2"))

	data, err = os.ReadFile(configPath)
	testza.AssertNoError(t, err)
	testza.AssertNoError(t, json.Unmarshal(data, &cfg))

	auths = cfg["auths"].(map[string]interface{})

	// registry1 still intact
	reg1 = auths["registry1.example.com"].(map[string]interface{})
	decoded1After, err := base64.StdEncoding.DecodeString(reg1["auth"].(string))
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "user1:pass1", string(decoded1After))

	// registry2 added
	reg2 := auths["registry2.example.com"].(map[string]interface{})
	decoded2, err := base64.StdEncoding.DecodeString(reg2["auth"].(string))
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "user2:pass2", string(decoded2))
}

func fileExistsRemote(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
