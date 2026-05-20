package helm

import "regexp"

// SecretKeyPattern is the case-insensitive regex used to identify values map
// keys that should be masked before display or diffing. The same pattern is
// re-implemented client-side in Phase 6.
const SecretKeyPattern = `(?i)(password|token|secret|key|cert|credential|apikey|passphrase)`

// secretKeyRE is the compiled form of SecretKeyPattern.
var secretKeyRE = regexp.MustCompile(SecretKeyPattern)

// MaskValue is the placeholder substituted for secret-like values.
const MaskValue = "••••••••"

// MaskValues walks a values map and replaces any value whose key matches
// SecretKeyPattern with MaskValue. Nested maps and slices of maps are walked
// recursively. The input is mutated in place and returned for chaining.
func MaskValues(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			if secretKeyRE.MatchString(k) {
				t[k] = MaskValue
				continue
			}
			t[k] = MaskValues(child)
		}
		return t
	case []any:
		for i, child := range t {
			t[i] = MaskValues(child)
		}
		return t
	default:
		return v
	}
}
