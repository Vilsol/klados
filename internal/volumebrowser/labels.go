package volumebrowser

import (
	"os"
	"os/user"
	"strings"
)

// SanitizeLabelValue normalizes an arbitrary string so it meets the Kubernetes
// label value constraints: [a-z0-9A-Z]([-_.a-z0-9A-Z]*[a-z0-9A-Z])? up to 63
// chars. Anything invalid becomes '-'; leading/trailing non-alphanumerics are
// trimmed. Returns "unknown" when the result would otherwise be empty.
func SanitizeLabelValue(s string) string {
	s = strings.ToLower(s)
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' {
			b = append(b, c)
		} else {
			b = append(b, '-')
		}
	}
	// Trim leading/trailing non-alphanumerics.
	start, end := 0, len(b)
	for start < end && !isAlnum(b[start]) {
		start++
	}
	for end > start && !isAlnum(b[end-1]) {
		end--
	}
	b = b[start:end]
	if len(b) > 63 {
		b = b[:63]
		// Re-trim in case truncation left trailing non-alnum.
		for len(b) > 0 && !isAlnum(b[len(b)-1]) {
			b = b[:len(b)-1]
		}
	}
	if len(b) == 0 {
		return "unknown"
	}
	return string(b)
}

func isAlnum(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
}

// CurrentHostLabel returns a sanitized label value for the current hostname.
func CurrentHostLabel() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return SanitizeLabelValue(h)
}

// CurrentUserLabel returns a sanitized label value for the current user.
func CurrentUserLabel() string {
	if u, err := user.Current(); err == nil && u.Username != "" {
		return SanitizeLabelValue(u.Username)
	}
	return SanitizeLabelValue(os.Getenv("USER"))
}
