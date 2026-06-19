// Package routing derives the deterministic key used to resolve a pipeline from
// the plaintext name carried in a webhook, without the server decrypting names.
//
// Pipeline names are stored end-to-end encrypted, so the server cannot match an
// incoming name against them. Instead, both the CLI (when creating a pipeline)
// and the server (when a project-scoped token posts a run) hash the normalized
// name the same way and match on that.
//
// Privacy note: the key is a hash of a usually low-entropy name (e.g. "CI"), so
// it is not a secret. It deliberately trades a little name privacy for the
// convenience of one project-wide token. Pipeline-bound tokens never use it.
package routing

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// Key returns the routing key for a pipeline name. Normalization (trim +
// lowercase) makes "CI", "ci" and " CI " resolve to the same pipeline. An empty
// or whitespace-only name yields "" (no routing key).
func Key(name string) string {
	norm := strings.ToLower(strings.TrimSpace(name))
	if norm == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(norm))
	return hex.EncodeToString(sum[:])
}
