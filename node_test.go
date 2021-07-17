package art

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLeafValue(t *testing.T) {
	leafNode := newLeafNode([]byte("foo"), "foo")

	if leafNode.Value() != "foo" {
		t.Error("Unexpected value for leafNode node")
	}

}

func TestNodeAddChild(t *testing.T) {
	nodes := []*artNode{newNode4(), newNode16(), newNode48(), newNode256()}

	for node := range nodes {
		n := nodes[node]

		for i := 0; i < n.maxSize(); i++ {
			newChild := newLeafNode([]byte{byte(i)}, byte(i))
			n.addChild(byte(i), newChild)
		}

		for i := 0; i < n.maxSize(); i++ {
			x := *(n.findChild(byte(i)))
			if x == nil {
				t.Error("Could not find child as expected")
			} else if x.Value() != byte(i) {
				t.Error("Child value does not match as expected")
			}
		}
	}
}

func TestIndexForAllNodeTypes(t *testing.T) {
	nodes := []*artNode{newNode4(), newNode16(), newNode48(), newNode256()}

	for node := range nodes {
		n := nodes[node]

		for i := 0; i < n.maxSize(); i++ {
			newChild := newLeafNode([]byte{byte(i)}, byte(i))
			n.addChild(byte(i), newChild)
		}

		for i := 0; i < n.maxSize(); i++ {
			if n.nodeType == Node48 {
				if n.index(byte(i)) != i+1 {
					t.Error("Unexpected value for Index function")
				}
			} else {
				if n.index(byte(i)) != i {
					t.Error("Unexpected value for Index function")
				}
			}

		}
	}
}

func TestArtNode4AddChild1AndFindChild(t *testing.T) {
	n := newNode4()
	n2 := newNode4()
	n.addChild('a', n2)

	assert.Equal(t, 1, n.node().size)

	x := *(n.findChild('a'))
	assert.Equal(t, n2, x)
}

func TestArtNode4AddChildTwicePreserveSorted(t *testing.T) {
	n := newNode4()
	n2 := newNode4()
	n3 := newNode4()
	n.addChild('b', n2)
	n.addChild('a', n3)

	if n.node().size < 2 {
		t.Error("Size is incorrect after adding one child to empty Node4")
	}

	if n.node4().keys[0] != 'a' {
		t.Error("Unexpected key value for first key index")
	}

	if n.node4().keys[1] != 'b' {
		t.Error("Unexpected key value for second key index")
	}
}

func TestArtNode4AddChild4PreserveSorted(t *testing.T) {
	n := newNode4()

	for i := 4; i > 0; i-- {
		n.addChild(byte(i), newNode4())
	}

	if n.node4().size < 4 {
		t.Error("Size is incorrect after adding one child to empty Node4")
	}

	expectedKeys := []byte{1, 2, 3, 4}
	if bytes.Compare(n.node4().keys[:], expectedKeys) != 0 {
		t.Error("Unexpected key sequence")
	}
}

func TestGrow(t *testing.T) {
	nodes := []*artNode{newNode4(), newNode16(), newNode48()}
	expectedTypes := []NodeType{Node16, Node48, Node256}

	for i := range nodes {
		node := nodes[i]

		node.grow()
		if node.nodeType != expectedTypes[i] {
			t.Error("Unexpected node type after growing")
		}
	}
}

func TestShrink(t *testing.T) {
	nodes := []*artNode{newNode48()}
	expectedTypes := []NodeType{Node16}

	for i := range nodes {
		node := nodes[i]

		for j := 0; j < node.minSize(); j++ {
			if node.nodeType != Node4 {
				node.addChild(byte(i), newNode4())
			} else {
				node.addChild(byte(i), newLeafNode(nil, nil))
			}
		}

		node.shrink()
		if node.nodeType != expectedTypes[i] {
			t.Error("Unexpected node type after shrinking")
		}
	}
}

func TestNewLeafNode(t *testing.T) {
	key := []byte{'a', 'r', 't'}
	value := "tree"
	l := newLeafNode(key, value)

	if &l.leafNode().key == &key {
		t.Errorf("Address of key byte slices should not match.")
	}

	if bytes.Compare(l.leafNode().key, key) != 0 {
		t.Errorf("Expected key value to match the one supplied")
	}

	if l.leafNode().value != value {
		t.Errorf("Expected initial value to match the one supplied")
	}

	if l.nodeType != LeafNode {
		t.Errorf("Expected LeafNode to be of LeafNode type")
	}
}
