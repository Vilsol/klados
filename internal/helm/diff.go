package helm

import (
	"context"
	"errors"
	"fmt"
	"strings"

	chartutil "helm.sh/helm/v4/pkg/chart/common/util"
	release "helm.sh/helm/v4/pkg/release/v1"
	"sigs.k8s.io/yaml"
)

// ErrRevisionNotFound is returned when a requested revision cannot be located
// in the underlying Secrets.
var ErrRevisionNotFound = errors.New("helm: revision not found")

// RevisionDiff carries the three textual diffs surfaced to the UI.
type RevisionDiff struct {
	Values         string
	ComputedValues string
	Manifest       string
}

// DiffRevisions builds a per-section unified diff between two revisions of
// the same release. Values are masked before diffing so secrets never leak
// into the UI.
func (b *Backend) DiffRevisions(ctx context.Context, contextName, namespace, releaseName string, from, to int) (RevisionDiff, error) {
	secrets, _, err := b.secretLister.ListSecrets(ctx, contextName, namespace, "", "name="+releaseName)
	if err != nil {
		return RevisionDiff{}, fmt.Errorf("helm diff: %w", err)
	}
	flat, err := ReassembleContinuation(secrets)
	if err != nil {
		flat = secrets
	}
	byRev := map[int]*release.Release{}
	for i := range flat {
		rel, err := DecodeRelease(&flat[i])
		if err != nil {
			continue
		}
		byRev[rel.Version] = rel
	}
	relFrom, okFrom := byRev[from]
	if !okFrom {
		return RevisionDiff{}, fmt.Errorf("%w: %d", ErrRevisionNotFound, from)
	}
	relTo, okTo := byRev[to]
	if !okTo {
		return RevisionDiff{}, fmt.Errorf("%w: %d", ErrRevisionNotFound, to)
	}
	out := RevisionDiff{}
	out.Values = diffYAMLValues(relFrom.Config, relTo.Config)
	out.ComputedValues = diffComputedValues(relFrom, relTo)
	out.Manifest = unifiedDiff(strings.TrimSpace(relFrom.Manifest), strings.TrimSpace(relTo.Manifest))
	return out, nil
}

func diffYAMLValues(a, b map[string]any) string {
	am := MaskValues(deepCopyMap(a))
	bm := MaskValues(deepCopyMap(b))
	aYAML, _ := yaml.Marshal(am)
	bYAML, _ := yaml.Marshal(bm)
	return unifiedDiff(strings.TrimRight(string(aYAML), "\n"), strings.TrimRight(string(bYAML), "\n"))
}

func diffComputedValues(a, b *release.Release) string {
	am := computeValues(a)
	bm := computeValues(b)
	aYAML, _ := yaml.Marshal(MaskValues(am))
	bYAML, _ := yaml.Marshal(MaskValues(bm))
	return unifiedDiff(strings.TrimRight(string(aYAML), "\n"), strings.TrimRight(string(bYAML), "\n"))
}

func computeValues(rel *release.Release) map[string]any {
	if rel == nil || rel.Chart == nil {
		return map[string]any{}
	}
	// Merge chart defaults with release config; ignore CoalesceValues errors
	// — diff is best-effort.
	merged, err := chartutil.CoalesceValues(rel.Chart, rel.Config)
	if err != nil {
		cp, _ := deepCopyMap(rel.Config).(map[string]any)
		if cp == nil {
			return map[string]any{}
		}
		return cp
	}
	return map[string]any(merged)
}

// deepCopyMap returns a deep copy of a values-style map so masking doesn't
// mutate the original.
func deepCopyMap(in any) any {
	switch t := in.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, v := range t {
			out[k] = deepCopyMap(v)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, v := range t {
			out[i] = deepCopyMap(v)
		}
		return out
	default:
		return t
	}
}

// unifiedDiff is a tiny line-by-line diff using LCS. The output mimics the
// classic unified-diff "+ "/"- " prefix style; we don't emit hunk headers
// because the UI renders side-by-side. Equal inputs return an empty string.
func unifiedDiff(a, b string) string {
	if a == b {
		return ""
	}
	la := splitLines(a)
	lb := splitLines(b)
	ops := lcsDiff(la, lb)
	var buf strings.Builder
	for _, op := range ops {
		switch op.kind {
		case 0:
			buf.WriteString("  ")
			buf.WriteString(op.line)
			buf.WriteByte('\n')
		case -1:
			buf.WriteString("- ")
			buf.WriteString(op.line)
			buf.WriteByte('\n')
		case 1:
			buf.WriteString("+ ")
			buf.WriteString(op.line)
			buf.WriteByte('\n')
		}
	}
	return buf.String()
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

type diffOp struct {
	kind int // -1 delete, 0 equal, 1 insert
	line string
}

func lcsDiff(a, b []string) []diffOp {
	n, m := len(a), len(b)
	// LCS table.
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}
	var ops []diffOp
	i, j := n, m
	for i > 0 && j > 0 {
		if a[i-1] == b[j-1] {
			ops = append(ops, diffOp{kind: 0, line: a[i-1]})
			i--
			j--
		} else if dp[i-1][j] >= dp[i][j-1] {
			ops = append(ops, diffOp{kind: -1, line: a[i-1]})
			i--
		} else {
			ops = append(ops, diffOp{kind: 1, line: b[j-1]})
			j--
		}
	}
	for i > 0 {
		ops = append(ops, diffOp{kind: -1, line: a[i-1]})
		i--
	}
	for j > 0 {
		ops = append(ops, diffOp{kind: 1, line: b[j-1]})
		j--
	}
	// Reverse.
	for l, r := 0, len(ops)-1; l < r; l, r = l+1, r-1 {
		ops[l], ops[r] = ops[r], ops[l]
	}
	return ops
}
