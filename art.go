package art

// NodeType - adaptive radix tree node type.
type NodeType uint8

// Types of node.
const (
	LeafNode NodeType = iota
	Node4
	Node16
	Node48
	Node256
)

// Key type.
type Key = []byte

// Value type.
type Value = interface{}

// Node interfaces
type Node interface {
	NodeType() NodeType
	Key() Key
	Value() Value
}

// Callback - callback function that is passed in Each.
type Callback func(node Node)

// Tree - adaptive radix tree interface.
type Tree interface {
	Insert(key Key, value Value)
	Search(key Key) (value Value)
	Delete(key Key) (deleted bool)
	Each(cb Callback)
	Size() int
}

// New - creates a new instance of adaptive radix tree.
func New() Tree {
	return newArt()
}
