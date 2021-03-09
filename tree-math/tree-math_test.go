package treeMath

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Precomputed answers for the tree on eleven elements:
//
//                                              X
//                      X
//          X                       X                       X
//    X           X           X           X           X
// X     X     X     X     X     X     X     X     X     X     X
// 0  1  2  3  4  5  6  7  8  9  a  b  c  d  e  f 10 11 12 13 14
var (
	aRoot = []NodeIndex{0x00, 0x01, 0x03, 0x03, 0x07, 0x07, 0x07, 0x07, 0x0f, 0x0f, 0x0f}

	aNil     = NodeIndex(0xffffffff)
	aN       = LeafCount(0x0b)
	index    = []NodeIndex{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14}
	aLog2    = []NodeIndex{0x00, 0x00, 0x01, 0x01, 0x02, 0x02, 0x02, 0x02, 0x03, 0x03, 0x03, 0x03, 0x03, 0x03, 0x03, 0x03, 0x04, 0x04, 0x04, 0x04, 0x04}
	aLevel   = []NodeIndex{0x00, 0x01, 0x00, 0x02, 0x00, 0x01, 0x00, 0x03, 0x00, 0x01, 0x00, 0x02, 0x00, 0x01, 0x00, 0x04, 0x00, 0x01, 0x00, 0x02, 0x00}
	aLeft    = []NodeIndex{0x00, 0x00, 0x02, 0x01, 0x04, 0x04, 0x06, 0x03, 0x08, 0x08, 0x0a, 0x09, 0x0c, 0x0c, 0x0e, 0x07, 0x10, 0x10, 0x12, 0x11, 0x14}
	aRight   = []NodeIndex{0x00, 0x02, 0x02, 0x05, 0x04, 0x06, 0x06, 0x0b, 0x08, 0x0a, 0x0a, 0x0d, 0x0c, 0x0e, 0x0e, 0x13, 0x10, 0x12, 0x12, 0x14, 0x14}
	aParent  = []NodeIndex{0x01, 0x03, 0x01, 0x07, 0x05, 0x03, 0x05, 0x0f, 0x09, 0x0b, 0x09, 0x07, 0x0d, 0x0b, 0x0d, 0x0f, 0x11, 0x13, 0x11, 0x0f, 0x13}
	aSibling = []NodeIndex{0x02, 0x05, 0x00, 0x0b, 0x06, 0x01, 0x04, 0x13, 0x0a, 0x0d, 0x08, 0x03, 0x0e, 0x09, 0x0c, 0x0f, 0x12, 0x14, 0x10, 0x07, 0x11}
)

func TestSizeProperties(t *testing.T) {
	for n := LeafCount(1); n < aN; n += 1 {
		if Root(n) != aRoot[n-1] {
			t.Fatalf("Root mismatch: %v != %v", Root(n), aRoot[n-1])
		}
	}
}

func TestNodeRelations(t *testing.T) {
	run := func(label string, f func(x NodeIndex) *NodeIndex, a []NodeIndex) {
		for i, x := range index {
			if a[i] == aNil {
				require.Equal(t, f(x), nil)
				continue
			}

			require.NotNil(t, f(x))
			require.Equal(t, *f(x), a[i])
		}
	}

	run("log2", func(x NodeIndex) *NodeIndex { out := NodeIndex(log2(NodeCount(x))); return &out }, aLog2)
	run("level", func(x NodeIndex) *NodeIndex { out := NodeIndex(level(x)); return &out }, aLevel)
	run("left", Left, aLeft)
	run("right", func(x NodeIndex) *NodeIndex { return Right(x, aN) }, aRight)
	run("parent", func(x NodeIndex) *NodeIndex { return Parent(x, aN) }, aParent)
	run("sibling", func(x NodeIndex) *NodeIndex { return Sibling(x, aN) }, aSibling)
}
