// continuation handles chunked release Secrets named
// sh.helm.release.v1.<release>.v<rev>.<chunk>. Helm v4 does not currently
// chunk Secrets >1 MB; this code is kept for forward-compatibility against
// future Helm versions and for third-party tools that chunk.
package helm

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// ErrMissingChunk is returned when ReassembleContinuation encounters a chunked
// release whose chunk indices are not contiguous from 0.
var ErrMissingChunk = errors.New("helm: missing continuation chunk")

// chunkSuffix matches the optional ".N" trailing component of a chunked
// release secret name (e.g. "sh.helm.release.v1.foo.v3.2").
var chunkSuffix = regexp.MustCompile(`^(sh\.helm\.release\.v1\..+\.v\d+)\.(\d+)$`)

// ReassembleContinuation accepts a flat list of Helm release Secrets (possibly
// containing chunked continuation entries) and returns a new slice where
// chunks have been combined into single virtual Secrets. Chunk payloads are
// concatenated in chunk-index order before any base64-decode happens.
//
// Non-chunked secrets are passed through unchanged. The order of distinct
// releases in the output is stable but not specified.
func ReassembleContinuation(secrets []corev1.Secret) ([]corev1.Secret, error) {
	// Group by base key. Non-chunked secrets get a unique key so they pass
	// through as a single-chunk group.
	type chunk struct {
		idx int
		sec corev1.Secret
	}
	groups := make(map[string][]chunk)
	order := make([]string, 0, len(secrets))

	for _, s := range secrets {
		m := chunkSuffix.FindStringSubmatch(s.Name)
		if m == nil {
			// Non-chunked: pass through with a unique key.
			key := s.Namespace + "//" + s.Name
			if _, ok := groups[key]; !ok {
				order = append(order, key)
			}
			groups[key] = append(groups[key], chunk{idx: 0, sec: s})
			continue
		}
		base := s.Namespace + "//" + m[1]
		idx, err := strconv.Atoi(m[2])
		if err != nil {
			return nil, fmt.Errorf("chunk index %q: %w", m[2], err)
		}
		if _, ok := groups[base]; !ok {
			order = append(order, base)
		}
		groups[base] = append(groups[base], chunk{idx: idx, sec: s})
	}

	out := make([]corev1.Secret, 0, len(order))
	for _, key := range order {
		chunks := groups[key]
		if len(chunks) == 1 {
			out = append(out, chunks[0].sec)
			continue
		}
		sort.Slice(chunks, func(i, j int) bool { return chunks[i].idx < chunks[j].idx })
		// Validate contiguity from 0.
		for i, c := range chunks {
			if c.idx != i {
				return nil, fmt.Errorf("%w: %s chunk %d", ErrMissingChunk, key, i)
			}
		}
		// Build combined Secret using chunk 0's metadata, then concatenate
		// release payloads. The base name has the chunk suffix stripped.
		base := chunks[0].sec
		baseName := strings.TrimSuffix(base.Name, fmt.Sprintf(".%d", chunks[0].idx))
		base.Name = baseName
		var combined []byte
		for _, c := range chunks {
			combined = append(combined, c.sec.Data["release"]...)
		}
		if base.Data == nil {
			base.Data = map[string][]byte{}
		}
		base.Data["release"] = combined
		out = append(out, base)
	}
	return out, nil
}
