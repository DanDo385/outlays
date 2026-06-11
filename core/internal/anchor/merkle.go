// Package anchor computes the D31 Merkle root over a run's fact hashes and submits it to
// the on-chain AnchorRegistry (task S12). The construction is specified normatively in
// ARCHITECTURE.md D31 so third parties can reproduce roots without reading this code:
//
//	leaves    = the run's fact_hash values (32-byte SHA-256, hex), decoded to raw bytes,
//	            sorted ascending bytewise, duplicates rejected
//	leafHash  = SHA-256(0x00 || leaf)
//	nodeHash  = SHA-256(0x01 || left || right)
//	pairing   = consecutive pairs left-to-right per level; a trailing odd node is
//	            promoted unchanged to the next level
//	root      = the single remaining node; a one-fact run's root is its leafHash;
//	            empty runs are refused
//
// SHA-256 throughout, consistent with every other hash in the system (Hard Rule 3).
package anchor

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
)

const (
	leafPrefix = byte(0x00)
	nodePrefix = byte(0x01)
)

// Root computes the D31 Merkle root over a set of fact hashes (64-char lowercase hex).
// Input order is irrelevant (leaves are sorted); duplicates and malformed hashes error.
func Root(factHashes []string) ([32]byte, error) {
	var zero [32]byte
	if len(factHashes) == 0 {
		return zero, fmt.Errorf("refusing to anchor an empty fact set")
	}

	leaves := make([][]byte, 0, len(factHashes))
	for _, h := range factHashes {
		raw, err := hex.DecodeString(h)
		if err != nil || len(raw) != 32 || h != hex.EncodeToString(raw) {
			return zero, fmt.Errorf("fact hash %q is not 64-char lowercase hex", h)
		}
		leaves = append(leaves, raw)
	}
	sort.Slice(leaves, func(i, j int) bool { return bytes.Compare(leaves[i], leaves[j]) < 0 })
	for i := 1; i < len(leaves); i++ {
		if bytes.Equal(leaves[i-1], leaves[i]) {
			return zero, fmt.Errorf("duplicate fact hash %x", leaves[i])
		}
	}

	level := make([][32]byte, len(leaves))
	for i, l := range leaves {
		level[i] = sha256.Sum256(append([]byte{leafPrefix}, l...))
	}
	for len(level) > 1 {
		next := make([][32]byte, 0, (len(level)+1)/2)
		for i := 0; i+1 < len(level); i += 2 {
			buf := make([]byte, 0, 65)
			buf = append(buf, nodePrefix)
			buf = append(buf, level[i][:]...)
			buf = append(buf, level[i+1][:]...)
			next = append(next, sha256.Sum256(buf))
		}
		if len(level)%2 == 1 {
			next = append(next, level[len(level)-1]) // odd node promoted unchanged
		}
		level = next
	}
	return level[0], nil
}

// RootHex returns the D31 root as a 0x-prefixed hex string.
func RootHex(factHashes []string) (string, error) {
	r, err := Root(factHashes)
	if err != nil {
		return "", err
	}
	return "0x" + hex.EncodeToString(r[:]), nil
}
