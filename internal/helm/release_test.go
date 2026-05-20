package helm

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/MarvinJWendt/testza"
	"helm.sh/helm/v4/pkg/release/common"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

func TestDecodeRelease_RoundTrip(t *testing.T) {
	s := makeReleaseSecret(t, "myapp", "default", 3, common.StatusDeployed, map[string]any{"replicas": 2}, "")
	rel, err := DecodeRelease(&s)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "myapp", rel.Name)
	testza.AssertEqual(t, 3, rel.Version)
	testza.AssertEqual(t, "myapp-chart", rel.Chart.Metadata.Name)
	testza.AssertEqual(t, common.StatusDeployed, rel.Info.Status)
}

func TestDecodeRelease_FromFixture(t *testing.T) {
	path := filepath.Join("testdata", "release-deployed.secret.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture not generated; run `go test -run TestGenerateFixtures -update`: %v", err)
	}
	var s corev1.Secret
	if err := yaml.Unmarshal(raw, &s); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	rel, err := DecodeRelease(&s)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "deployed-app", rel.Name)
	testza.AssertEqual(t, common.StatusDeployed, rel.Info.Status)
}

func TestDecodeRelease_MissingPayload(t *testing.T) {
	s := corev1.Secret{Type: ReleaseSecretType, Data: map[string][]byte{}}
	_, err := DecodeRelease(&s)
	testza.AssertTrue(t, errors.Is(err, ErrMissingPayload))
}

func TestDecodeRelease_WrongType(t *testing.T) {
	s := corev1.Secret{Type: corev1.SecretTypeOpaque, Data: map[string][]byte{"release": []byte("x")}}
	_, err := DecodeRelease(&s)
	testza.AssertTrue(t, errors.Is(err, ErrWrongType))
}

func TestDecodeRelease_BadBase64(t *testing.T) {
	s := corev1.Secret{Type: ReleaseSecretType, Data: map[string][]byte{"release": []byte("!!!not-base64!!!")}}
	_, err := DecodeRelease(&s)
	testza.AssertNotNil(t, err)
	testza.AssertContains(t, err.Error(), "base64")
}

func TestDecodeRelease_BadGzip(t *testing.T) {
	// 0x1f 0x8b 0xff — gzip magic but invalid body
	bad := []byte{0x1f, 0x8b, 0xff, 0x00}
	encoded := base64.StdEncoding.EncodeToString(bad)
	s := corev1.Secret{Type: ReleaseSecretType, Data: map[string][]byte{"release": []byte(encoded)}}
	_, err := DecodeRelease(&s)
	testza.AssertNotNil(t, err)
	testza.AssertContains(t, err.Error(), "gzip")
}

func TestDecodeRelease_BadJSON(t *testing.T) {
	// gzip("not json")
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, _ = w.Write([]byte("not valid json"))
	_ = w.Close()
	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	s := corev1.Secret{Type: ReleaseSecretType, Data: map[string][]byte{"release": []byte(encoded)}}
	_, err := DecodeRelease(&s)
	testza.AssertNotNil(t, err)
	testza.AssertContains(t, err.Error(), "json")
}

func TestDecodeRelease_TooLarge(t *testing.T) {
	// Build > 50 MiB of decompressed payload.
	big := bytes.Repeat([]byte{'a'}, (MaxDecompressed + 1))
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, _ = w.Write(big)
	_ = w.Close()
	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	s := corev1.Secret{Type: ReleaseSecretType, Data: map[string][]byte{"release": []byte(encoded)}}
	_, err := DecodeRelease(&s)
	testza.AssertTrue(t, errors.Is(err, ErrTooLarge))
}
