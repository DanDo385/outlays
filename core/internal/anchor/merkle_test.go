package anchor

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func hexHash(b byte) string {
	raw := make([]byte, 32)
	for i := range raw {
		raw[i] = b
	}
	return hex.EncodeToString(raw)
}

func leaf(t *testing.T, hexStr string) [32]byte {
	t.Helper()
	raw, err := hex.DecodeString(hexStr)
	if err != nil {
		t.Fatal(err)
	}
	return sha256.Sum256(append([]byte{0x00}, raw...))
}

func node(l, r [32]byte) [32]byte {
	buf := append([]byte{0x01}, l[:]...)
	return sha256.Sum256(append(buf, r[:]...))
}

func TestSingleLeafRootIsLeafHash(t *testing.T) {
	h := hexHash(0xaa)
	got, err := Root([]string{h})
	if err != nil {
		t.Fatal(err)
	}
	if got != leaf(t, h) {
		t.Error("single-leaf root must equal SHA-256(0x00 || leaf)")
	}
}

func TestTwoLeavesSortedPairing(t *testing.T) {
	a, b := hexHash(0x11), hexHash(0x22)
	want := node(leaf(t, a), leaf(t, b))
	// Input order must not matter: leaves sort ascending bytewise.
	for _, in := range [][]string{{a, b}, {b, a}} {
		got, err := Root(in)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("two-leaf root mismatch for input %v", in)
		}
	}
}

func TestOddNodePromotion(t *testing.T) {
	a, b, c := hexHash(0x11), hexHash(0x22), hexHash(0x33)
	// Level 0: [leaf(a), leaf(b), leaf(c)] -> level 1: [node(la,lb), leaf(c) promoted]
	// Root: node(node(la,lb), leaf(c)) — the odd node is promoted unchanged, not duplicated.
	want := node(node(leaf(t, a), leaf(t, b)), leaf(t, c))
	got, err := Root([]string{c, a, b})
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Error("three-leaf root must promote the trailing odd node unchanged")
	}
}

func TestFiveLeavesStructure(t *testing.T) {
	hs := []string{hexHash(0x01), hexHash(0x02), hexHash(0x03), hexHash(0x04), hexHash(0x05)}
	l := make([][32]byte, 5)
	for i, h := range hs {
		l[i] = leaf(t, h)
	}
	// level1: [n(0,1), n(2,3), l4]   level2: [n(n01,n23), l4]   root: n(level2...)
	want := node(node(node(l[0], l[1]), node(l[2], l[3])), l[4])
	got, err := Root(hs)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Error("five-leaf structure mismatch")
	}
}

func TestRejectsEmptyDuplicateMalformed(t *testing.T) {
	if _, err := Root(nil); err == nil {
		t.Error("empty set must be refused")
	}
	if _, err := Root([]string{hexHash(0xaa), hexHash(0xaa)}); err == nil {
		t.Error("duplicate hashes must be refused")
	}
	for _, bad := range []string{"zz", hexHash(0xaa)[:62], strings.ToUpper(hexHash(0xaa)), "0x" + hexHash(0xaa)} {
		if _, err := Root([]string{bad}); err == nil {
			t.Errorf("malformed hash %q must be refused", bad)
		}
	}
}

func TestRootHexFormat(t *testing.T) {
	s, err := RootHex([]string{hexHash(0xab)})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(s, "0x") || len(s) != 66 {
		t.Errorf("RootHex = %q", s)
	}
}
