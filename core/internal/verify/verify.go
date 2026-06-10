// Package verify holds hashing and re-derivation: JCS (RFC 8785) canonical-JSON hashing and
// the deterministic resultHash recomputation used to verify adapter output by re-derivation
// (ARCHITECTURE.md Section 4, Decision D15). Kept in lockstep with the SDKs.
package verify

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"
)

// VolatileFactFields are dropped before hashing a fact set (DB-assigned, non-deterministic).
var VolatileFactFields = []string{"factId", "runId", "insertedAt"}

// JCSSha256 returns the SHA-256 hex over the RFC 8785 canonical form of the given JSON bytes.
func JCSSha256(raw []byte) (string, error) {
	canon, err := jsoncanonicalizer.Transform(raw)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canon)
	return hex.EncodeToString(sum[:]), nil
}

// RecomputeResultHash mirrors the SDKs: drop volatile fields from each fact, sort by factHash
// ascending, then JCS + SHA-256 over the array.
func RecomputeResultHash(facts []json.RawMessage) (string, error) {
	type factMap map[string]json.RawMessage
	parsed := make([]factMap, 0, len(facts))
	for _, raw := range facts {
		var m factMap
		if err := json.Unmarshal(raw, &m); err != nil {
			return "", err
		}
		for _, k := range VolatileFactFields {
			delete(m, k)
		}
		parsed = append(parsed, m)
	}
	sort.Slice(parsed, func(i, j int) bool {
		return string(parsed[i]["factHash"]) < string(parsed[j]["factHash"])
	})
	arr, err := json.Marshal(parsed)
	if err != nil {
		return "", err
	}
	return JCSSha256(arr)
}
