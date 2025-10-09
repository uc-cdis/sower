package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
)

func filter(scs []SowerConfig, test func(SowerConfig) bool) (ret []SowerConfig) {
	for _, s := range scs {
		if test(s) {
			ret = append(ret, s)
		}
	}
	return
}

var (
	// valid: alphanum, '-', '_', '.', must start/end alnum, <= 63 chars
	labelAllowed = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)
	trimEnds     = regexp.MustCompile(`^[^A-Za-z0-9]+|[^A-Za-z0-9]+$`)
	collapseDash = regexp.MustCompile(`[-._]{2,}`)
)

func sanitizeLabelValue(raw string) string {
	if raw == "" {
		return "id-" + shortHash("empty")
	}
	s := labelAllowed.ReplaceAllString(raw, "-")
	s = collapseDash.ReplaceAllString(s, "-")
	s = trimEnds.ReplaceAllString(s, "")
	if len(s) > 63 {
		s = s[:63]
	}
	if s == "" {
		return "id-" + shortHash(raw)
	}
	return s
}

func shortHash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])[:16]
}