package mls

import (
	"reflect"
	"testing"

	"github.com/bifurcation/mint/syntax"
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
	aRoot = []nodeIndex{0x00, 0x01, 0x03, 0x03, 0x07, 0x07, 0x07, 0x07, 0x0f, 0x0f, 0x0f}

	aN       = leafCount(0x0b)
	index    = []nodeIndex{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14}
	aLog2    = []nodeIndex{0x00, 0x00, 0x01, 0x01, 0x02, 0x02, 0x02, 0x02, 0x03, 0x03, 0x03, 0x03, 0x03, 0x03, 0x03, 0x03, 0x04, 0x04, 0x04, 0x04, 0x04}
	aLevel   = []nodeIndex{0x00, 0x01, 0x00, 0x02, 0x00, 0x01, 0x00, 0x03, 0x00, 0x01, 0x00, 0x02, 0x00, 0x01, 0x00, 0x04, 0x00, 0x01, 0x00, 0x02, 0x00}
	aLeft    = []nodeIndex{0x00, 0x00, 0x02, 0x01, 0x04, 0x04, 0x06, 0x03, 0x08, 0x08, 0x0a, 0x09, 0x0c, 0x0c, 0x0e, 0x07, 0x10, 0x10, 0x12, 0x11, 0x14}
	aRight   = []nodeIndex{0x00, 0x02, 0x02, 0x05, 0x04, 0x06, 0x06, 0x0b, 0x08, 0x0a, 0x0a, 0x0d, 0x0c, 0x0e, 0x0e, 0x13, 0x10, 0x12, 0x12, 0x14, 0x14}
	aParent  = []nodeIndex{0x01, 0x03, 0x01, 0x07, 0x05, 0x03, 0x05, 0x0f, 0x09, 0x0b, 0x09, 0x07, 0x0d, 0x0b, 0x0d, 0x0f, 0x11, 0x13, 0x11, 0x0f, 0x13}
	aSibling = []nodeIndex{0x02, 0x05, 0x00, 0x0b, 0x06, 0x01, 0x04, 0x13, 0x0a, 0x0d, 0x08, 0x03, 0x0e, 0x09, 0x0c, 0x0f, 0x12, 0x14, 0x10, 0x07, 0x11}

	aDirpath = [][]nodeIndex{
		{0x01, 0x03, 0x07, 0x0f},
		{0x03, 0x07, 0x0f},
		{0x01, 0x03, 0x07, 0x0f},
		{0x07, 0x0f},
		{0x05, 0x03, 0x07, 0x0f},
		{0x03, 0x07, 0x0f},
		{0x05, 0x03, 0x07, 0x0f},
		{0x0f},
		{0x09, 0x0b, 0x07, 0x0f},
		{0x0b, 0x07, 0x0f},
		{0x09, 0x0b, 0x07, 0x0f},
		{0x07, 0x0f},
		{0x0d, 0x0b, 0x07, 0x0f},
		{0x0b, 0x07, 0x0f},
		{0x0d, 0x0b, 0x07, 0x0f},
		{},
		{0x11, 0x13, 0x0f},
		{0x13, 0x0f},
		{0x11, 0x13, 0x0f},
		{0x0f},
		{0x13, 0x0f},
	}
	aCopath = [][]nodeIndex{
		{0x02, 0x05, 0x0b, 0x13},
		{0x05, 0x0b, 0x13},
		{0x00, 0x05, 0x0b, 0x13},
		{0x0b, 0x13},
		{0x06, 0x01, 0x0b, 0x13},
		{0x01, 0x0b, 0x13},
		{0x04, 0x01, 0x0b, 0x13},
		{0x13},
		{0x0a, 0x0d, 0x03, 0x13},
		{0x0d, 0x03, 0x13},
		{0x08, 0x0d, 0x03, 0x13},
		{0x03, 0x13},
		{0x0e, 0x09, 0x03, 0x13},
		{0x09, 0x03, 0x13},
		{0x0c, 0x09, 0x03, 0x13},
		{},
		{0x12, 0x14, 0x07},
		{0x14, 0x07},
		{0x10, 0x14, 0x07},
		{0x07},
		{0x11, 0x07},
	}

	aAncestor = [][]nodeIndex{
		{0x01, 0x03, 0x03, 0x07, 0x07, 0x07, 0x07, 0x0f, 0x0f, 0x0f},
		{0x03, 0x03, 0x07, 0x07, 0x07, 0x07, 0x0f, 0x0f, 0x0f},
		{0x05, 0x07, 0x07, 0x07, 0x07, 0x0f, 0x0f, 0x0f},
		{0x07, 0x07, 0x07, 0x07, 0x0f, 0x0f, 0x0f},
		{0x09, 0x0b, 0x0b, 0x0f, 0x0f, 0x0f},
		{0x0b, 0x0b, 0x0f, 0x0f, 0x0f},
		{0x0d, 0x0f, 0x0f, 0x0f},
		{0x0f, 0x0f, 0x0f},
		{0x11, 0x13},
		{0x13},
	}
)

func TestSizeProperties(t *testing.T) {
	for n := leafCount(1); n < aN; n += 1 {
		if root(n) != aRoot[n-1] {
			t.Fatalf("Root mismatch: %v != %v", root(n), aRoot[n-1])
		}
	}
}

func TestNodeRelations(t *testing.T) {
	run := func(label string, f func(x nodeIndex) nodeIndex, a []nodeIndex) {
		for i, x := range index {
			if f(x) != a[i] {
				t.Fatalf("Relation test failure: %s @ 0x%02x: %v != %v", label, x, f(x), a[i])
			}
		}
	}

	run("log2", func(x nodeIndex) nodeIndex { return nodeIndex(log2(nodeCount(x))) }, aLog2)
	run("level", func(x nodeIndex) nodeIndex { return nodeIndex(level(x)) }, aLevel)
	run("left", left, aLeft)
	run("right", func(x nodeIndex) nodeIndex { return right(x, aN) }, aRight)
	run("parent", func(x nodeIndex) nodeIndex { return parent(x, aN) }, aParent)
	run("sibling", func(x nodeIndex) nodeIndex { return sibling(x, aN) }, aSibling)
}

func TestPaths(t *testing.T) {
	run := func(label string, f func(x nodeIndex, n leafCount) []nodeIndex, a [][]nodeIndex) {
		for i, x := range index {
			if !reflect.DeepEqual(f(x, aN), a[i]) {
				t.Fatalf("Path test failure: %s @ 0x%02x: %v != %v", label, x, f(x, aN), a[i])
			}
		}
	}

	run("dirpath", dirpath, aDirpath)
	run("copath", copath, aCopath)
}

func TestAncestor(t *testing.T) {
	for l := leafIndex(0); l < leafIndex(aN-1); l += 1 {
		for r := l + 1; r < leafIndex(aN); r += 1 {
			answer := aAncestor[l][r-l-1]
			lr := ancestor(l, r)
			rl := ancestor(r, l)

			if lr != answer {
				t.Fatalf("Incorrect ancestor: %d %d => %d != %d", l, r, lr, answer)
			}

			if lr != answer {
				t.Fatalf("Asymmetric ancestor: %d %d => %d != %d", l, r, rl, lr)
			}
		}
	}
}

///
/// Test Vectors
///

type TreeMathTestVectors struct {
	NumLeaves leafCount
	Root      []nodeIndex `tls:"head=4"`
	Left      []nodeIndex `tls:"head=4"`
	Right     []nodeIndex `tls:"head=4"`
	Parent    []nodeIndex `tls:"head=4"`
	Sibling   []nodeIndex `tls:"head=4"`
}

func generateTreeMathVectors(t *testing.T) []byte {
	numLeaves := leafCount(255)
	numNodes := nodeWidth(numLeaves)
	tv := TreeMathTestVectors{
		NumLeaves: numLeaves,
		Root:      make([]nodeIndex, numLeaves),
		Left:      make([]nodeIndex, numNodes),
		Right:     make([]nodeIndex, numNodes),
		Parent:    make([]nodeIndex, numNodes),
		Sibling:   make([]nodeIndex, numNodes),
	}

	for i := range tv.Root {
		tv.Root[i] = root(leafCount(i + 1))
	}

	for i := range tv.Left {
		tv.Left[i] = left(nodeIndex(i))
		tv.Right[i] = right(nodeIndex(i), numLeaves)
		tv.Parent[i] = parent(nodeIndex(i), numLeaves)
		tv.Sibling[i] = sibling(nodeIndex(i), numLeaves)
	}

	vec, err := syntax.Marshal(tv)
	require.Nil(t, err)
	return vec
}

func verifyTreeMathVectors(t *testing.T, data []byte) {
	var tv TreeMathTestVectors
	_, err := syntax.Unmarshal(data, &tv)
	require.Nil(t, err)

	tvLen := int(nodeWidth(tv.NumLeaves))
	if len(tv.Root) != int(tv.NumLeaves) || len(tv.Left) != tvLen ||
		len(tv.Right) != tvLen || len(tv.Parent) != tvLen || len(tv.Sibling) != tvLen {
		t.Fatalf("Malformed tree math test vectors: Incorrect vector sizes")
	}

	for i := range tv.Root {
		require.Equal(t, tv.Root[i], root(leafCount(i+1)))
	}

	for i := range tv.Left {
		require.Equal(t, tv.Left[i], left(nodeIndex(i)))
		require.Equal(t, tv.Right[i], right(nodeIndex(i), tv.NumLeaves))
		require.Equal(t, tv.Parent[i], parent(nodeIndex(i), tv.NumLeaves))
		require.Equal(t, tv.Sibling[i], sibling(nodeIndex(i), tv.NumLeaves))
	}
}

func TestTreeMathErrorCases(t *testing.T) {
	_, err := toLeafIndex(0x03)
	require.NotNil(t, err)
}
