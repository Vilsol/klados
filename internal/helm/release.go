package helm

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	release "helm.sh/helm/v4/pkg/release/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	// ReleaseSecretType is the Type field of Helm v3/v4 release Secrets.
	ReleaseSecretType = "helm.sh/release.v1"

	// MaxDecompressed is the hard cap for a single decompressed release payload.
	MaxDecompressed = 50 << 20 // 50 MiB
)

var (
	// ErrWrongType is returned by DecodeRelease when the Secret's Type field
	// does not match ReleaseSecretType.
	ErrWrongType = errors.New("helm: secret type mismatch")

	// ErrMissingPayload is returned when the Secret has no "release" data key.
	ErrMissingPayload = errors.New("helm: secret missing release data")

	// ErrTooLarge is returned when the decompressed payload would exceed MaxDecompressed.
	ErrTooLarge = fmt.Errorf("helm: decompressed payload exceeds %d bytes", MaxDecompressed)
)

// DecodeRelease decodes the `release` data field of a Helm release Secret into
// a *release.Release. It enforces a 50 MiB decompressed-size cap as a zip-bomb
// guard.
func DecodeRelease(s *corev1.Secret) (*release.Release, error) {
	if s.Type != ReleaseSecretType {
		return nil, fmt.Errorf("%w: %s", ErrWrongType, s.Type)
	}
	raw, ok := s.Data["release"]
	if !ok {
		return nil, ErrMissingPayload
	}
	decoded, err := base64.StdEncoding.DecodeString(string(raw))
	if err != nil {
		return nil, fmt.Errorf("base64: %w", err)
	}
	if len(decoded) >= 3 && decoded[0] == 0x1f && decoded[1] == 0x8b {
		gz, err := gzip.NewReader(bytes.NewReader(decoded))
		if err != nil {
			return nil, fmt.Errorf("gzip: %w", err)
		}
		defer gz.Close()
		decoded, err = io.ReadAll(io.LimitReader(gz, MaxDecompressed+1))
		if err != nil {
			return nil, fmt.Errorf("gunzip: %w", err)
		}
		if len(decoded) > MaxDecompressed {
			return nil, ErrTooLarge
		}
	}
	var rel release.Release
	if err := json.Unmarshal(decoded, &rel); err != nil {
		return nil, fmt.Errorf("json: %w", err)
	}
	return &rel, nil
}

// EncodeRelease encodes a *release.Release into the Secret data wire format
// (gzip + base64 of the JSON-marshalled release). Used by tests and fixture
// generation.
func EncodeRelease(rel *release.Release) ([]byte, error) {
	b, err := json.Marshal(rel)
	if err != nil {
		return nil, fmt.Errorf("json: %w", err)
	}
	var buf bytes.Buffer
	w, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, fmt.Errorf("gzip: %w", err)
	}
	if _, err := w.Write(b); err != nil {
		return nil, fmt.Errorf("gzip write: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("gzip close: %w", err)
	}
	encoded := make([]byte, base64.StdEncoding.EncodedLen(buf.Len()))
	base64.StdEncoding.Encode(encoded, buf.Bytes())
	return encoded, nil
}
