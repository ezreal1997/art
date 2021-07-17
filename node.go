package art

import (
	"bytes"
	"sort"
	"unsafe"
)

const (
	// Inner nodes of type Node4 must have between 2 and 4 children.
	node4Min = 2
	node4Max = 4

	// Inner nodes of type Node16 must have between 5 and 16 children.
	node16Min = 5
	node16Max = 16

	// Inner nodes of type Node48 must have between 17 and 48 children.
	node48Min = 17
	node48Max = 48

	// Inner nodes of type Node256 must have between 49 and 256 children.
	node256Min = 49
	node256Max = 256

	maxPrefixLen = 10
)

// nullNode represent for nil value, so as not to make redundant allocations.
var nullNode *artNode = nil

// node includes metadata of art tree node.
type node struct {
	size      int
	prefixLen int
	prefix    [maxPrefixLen]byte
}

// node4 is of type Node4
type node4 struct {
	node
	keys     [node4Max]byte
	children [node4Max]*artNode
}

// node16 is of type Node16
type node16 struct {
	node
	keys     [node16Max]byte
	children [node16Max]*artNode
}

// node48 is of type Node48
type node48 struct {
	node
	keys     [node256Max]byte        // keys[$(prefix_char)] = $(idx in children)
	children [node48Max + 1]*artNode // Do not use children[0] as 0 is the default value of keys[$(prefix_char)]
}

// node256 is of type Node256
type node256 struct {
	node
	children [node256Max]*artNode // children[$(char)] = $(child pointer)
}

// leafNode contains the real key value data.
type leafNode struct {
	key   Key
	value interface{}
}

// artNode is an embedded node type used for art.
type artNode struct {
	nodeType NodeType
	nodePtr  unsafe.Pointer
}

// // newLeafNode creates an embedded artNode of leafNode
func newLeafNode(key []byte, value interface{}) *artNode {
	newKey := make([]byte, len(key))
	copy(newKey, key)
	return &artNode{
		nodeType: LeafNode,
		nodePtr:  unsafe.Pointer(&leafNode{key: newKey, value: value}),
	}
}

// newNode4 creates an embedded artNode of node4
func newNode4() *artNode {
	return &artNode{nodeType: Node4, nodePtr: unsafe.Pointer(&node4{})}
}

// newNode16 creates an embedded artNode of node16
func newNode16() *artNode {
	return &artNode{nodeType: Node16, nodePtr: unsafe.Pointer(&node16{})}
}

// newNode48 creates an embedded artNode of node48
func newNode48() *artNode {
	return &artNode{nodeType: Node48, nodePtr: unsafe.Pointer(&node48{})}
}

// newNode256 creates an embedded artNode of node256
func newNode256() *artNode {
	return &artNode{nodeType: Node256, nodePtr: unsafe.Pointer(&node256{})}
}

// Key returns the key of the given node, or nil if it is not a leafNode.
func (n *artNode) Key() Key {
	if n.isLeaf() {
		return n.leafNode().key
	}
	return nil
}

// Value returns the value of the given node, or nil if it is not a leafNode.
func (n *artNode) Value() interface{} {
	if n.nodeType != LeafNode {
		return nil
	}
	return n.leafNode().value
}

// NodeType returns the nodeType of the given node
func (n *artNode) NodeType() NodeType {
	return n.nodeType
}

// isFull returns whether this particular artNode is full or not .
func (n *artNode) isFull() bool {
	return n.node().size == n.maxSize()
}

// isLeaf returns whether this particular artNode is a leafNode or not .
func (n *artNode) isLeaf() bool { return n.nodeType == LeafNode }

// isMatch returns whether the key stored in the leafNode matches the passed in key or not .
func (n *artNode) isMatch(key []byte) bool {
	if n.nodeType != LeafNode {
		return false
	}
	if len(n.leafNode().key) != len(key) {
		return false
	}
	return bytes.Compare(n.leafNode().key, key) == 0
}

// prefixMismatch returns the position of first byte that differ between the passed in key
// and the compressed path of the current node at the specified depth.
func (n *artNode) prefixMismatch(key []byte, depth int) int {
	var idx int

	var keyChar byte
	for idx = 0; idx < min(maxPrefixLen, n.node().prefixLen); idx++ {
		if depth+idx < 0 || depth+idx >= len(key) {
			keyChar = byte(0)
		} else {
			keyChar = key[depth+idx]
		}
		if keyChar != n.node().prefix[idx] {
			return idx
		}
	}

	if n.node().prefixLen > maxPrefixLen {
		minKey := n.minimum().leafNode().key
		for ; idx < n.node().prefixLen; idx++ {
			if key[depth+idx] != minKey[depth+idx] {
				return idx
			}
		}
	}

	return idx
}

// index returns the position of the given key byte's child pointer in the children array.
// If not found, return -1.
func (n *artNode) index(key byte) int {
	switch n.nodeType {
	case Node4:
		return bytes.IndexByte(n.node4().keys[:], key)
	case Node16:
		return bytes.IndexByte(n.node16().keys[:], key)
	case Node48:
		return int(n.node48().keys[key])
	case Node256:
		return int(key)
	}
	return -1
}

// findChild returns a pointer to the child that matches the passed in key,
// or nil if not present.
func (n *artNode) findChild(key byte) **artNode {
	if n == nil {
		return &nullNode
	}

	var idx int
	switch n.nodeType {
	case Node4, Node16, Node48:
		idx = n.index(key)
	case Node256:
		idx = int(key)
	}
	// Not found.
	if idx < 0 {
		return &nullNode
	}

	switch n.nodeType {
	case Node4:
		return &n.node4().children[idx]
	case Node16:
		return &n.node16().children[idx]
	case Node48:
		if idx == 0 { // children[0] is not used in Node48
			return &nullNode
		}
		return &n.node48().children[idx]
	case Node256:
		if n.node256().children[idx] == nil {
			return &nullNode
		}
		return &n.node256().children[idx]
	}

	return &nullNode
}

// addChild adds the passed in artNode to the current artNode's children at the specified key.
// The current node will grow if necessary when the insertion to take place.
func (n *artNode) addChild(key byte, node *artNode) {
	switch n.nodeType {
	case Node4:
		if n.isFull() {
			n.grow()
			n.addChild(key, node)
			break
		}
		n4 := n.node4()
		var idx int
		if n4.size == 0 {
			idx = 0
		} else {
			for idx = 0; idx < n4.size; idx++ {
				if key < n4.keys[idx] {
					break
				}
			}
		}
		for i := n4.size; i > idx; i-- {
			n4.keys[i] = n4.keys[i-1]
			n4.children[i] = n4.children[i-1]
		}
		n4.keys[idx] = key
		n4.children[idx] = node
		n4.size++
	case Node16:
		if n.isFull() {
			n.grow()
			n.addChild(key, node)
			break
		}
		n16 := n.node16()
		var idx int
		if n16.size == 0 {
			idx = 0
		} else {
			idx = sort.Search(n16.size, func(i int) bool {
				return key <= n16.keys[byte(i)]
			})
		}
		for i := n16.size; i > idx; i-- {
			n16.keys[i] = n16.keys[i-1]
			n16.children[i] = n16.children[i-1]
		}
		n16.keys[idx] = key
		n16.children[idx] = node
		n16.size++
	case Node48:
		if n.isFull() {
			n.grow()
			n.addChild(key, node)
			break
		}
		n48 := n.node48()
		idx := 1
		for n48.children[idx] != nil {
			idx++
		}
		n48.children[idx] = node
		n48.keys[key] = byte(idx)
		n48.size++
	case Node256:
		if n.isFull() {
			break
		}
		n.node256().children[key] = node
		n.node().size++
	}
}

// RemoveChild removes the child of the passed in key,
// and will shrink if it falls below its minimum size.
func (n *artNode) RemoveChild(key byte) {
	switch n.nodeType {
	case Node4:
		n4 := n.node4()
		idx := n.index(key)
		if idx < 0 {
			break
		}
		n4.keys[idx] = byte(0)
		n4.children[idx] = nil
		for i := idx; i < n4.size-1; i++ {
			n4.keys[i] = n4.keys[i+1]
			n4.children[i] = n4.children[i+1]
		}
		n4.keys[n4.size-1] = byte(0)
		n4.children[n4.size-1] = nil
		n4.size--
	case Node16:
		n16 := n.node16()
		idx := n.index(key)
		if idx < 0 {
			break
		}
		n16.keys[idx] = 0
		n16.children[idx] = nil
		for i := idx; i < n16.size-1; i++ {
			n16.keys[i] = n16.keys[i+1]
			n16.children[i] = n16.children[i+1]
		}
		n16.keys[n16.size-1] = 0
		n16.children[n16.size-1] = nil
		n16.size--
	case Node48:
		n48 := n.node48()
		idx := n.index(key)
		if idx <= 0 {
			break
		}
		n48.children[idx] = nil
		n48.keys[key] = byte(0)
		n48.size--
	case Node256:
		n256 := n.node256()
		n256.children[n.index(key)] = nil
		n256.size--
	}
	if n.node().size < n.minSize() {
		n.shrink()
	}
}

// grow upgrades the current artNode to contain more children.
func (n *artNode) grow() {
	switch n.nodeType {
	case Node4:
		newNode := newNode16()
		newNode.copyMeta(n)
		newNode16 := newNode.node16()
		n4 := n.node4()
		for i := 0; i < n4.size; i++ {
			newNode16.keys[i] = n4.keys[i]
			newNode16.children[i] = n4.children[i]
		}
		n.replaceWith(newNode)
	case Node16:
		newNode := newNode48()
		newNode.copyMeta(n)
		newNode48 := newNode.node48()
		n16 := n.node16()
		for i := 0; i < n16.size; i++ {
			newNode48.keys[n16.keys[i]] = byte(i + 1)
			newNode48.children[i+1] = n16.children[i]
		}
		n.replaceWith(newNode)
	case Node48:
		newNode := newNode256()
		newNode.copyMeta(n)
		newNode256 := newNode.node256()
		n48 := n.node48()
		for i := 0; i < len(n48.keys); i++ {
			if n48.keys[i] == byte(0) {
				continue
			}
			if n48.children[n48.keys[i]] != nil && n48.children[n48.keys[i]] != nullNode {
				newNode256.children[byte(i)] = n48.children[n48.keys[i]]
			}
		}
		n.replaceWith(newNode)
	case Node256:
		// Can not get bigger
	}
}

// shrink downgrades the current artNode to reduce the memory cost.
func (n *artNode) shrink() {
	switch n.nodeType {
	case Node4:
		n4 := n.node4()
		newNode := n4.children[0]
		if !newNode.isLeaf() {
			currentPrefixLen := n4.prefixLen
			if currentPrefixLen < maxPrefixLen {
				n4.prefix[currentPrefixLen] = n4.keys[0]
				currentPrefixLen++
			}
			if currentPrefixLen < maxPrefixLen {
				childPrefixLen := min(newNode.node().prefixLen, maxPrefixLen-currentPrefixLen)
				memcpy(n4.prefix[currentPrefixLen:], newNode.node().prefix[:], childPrefixLen)
				currentPrefixLen += childPrefixLen
			}
			memcpy(newNode.node().prefix[:], n4.prefix[:], min(currentPrefixLen, maxPrefixLen))
			newNode.node().prefixLen += n4.prefixLen + 1
		}
		n.replaceWith(newNode)
	case Node16:
		n16 := n.node16()
		newNode := newNode4()
		newNode.copyMeta(n)
		newNode4 := newNode.node4()
		newNode4.size = 0
		for i := 0; i < n16.size; i++ {
			newNode4.keys[newNode4.size] = n16.keys[i]
			newNode4.children[newNode4.size] = n16.children[i]
			newNode4.size++
		}
		n.replaceWith(newNode)
	case Node48:
		n48 := n.node48()
		newNode := newNode16()
		newNode.copyMeta(n)
		newNode16 := newNode.node16()
		newNode16.size = 0
		for i := 0; i < len(n48.keys); i++ {
			idx := n48.keys[byte(i)]
			if idx <= 0 {
				continue
			}
			newNode16.keys[newNode16.size] = byte(i)
			newNode16.children[newNode16.size] = n48.children[idx]
			newNode16.size++
		}
		n.replaceWith(newNode)
	case Node256:
		n256 := n.node256()
		newNode := newNode48()
		newNode.copyMeta(n)
		newNode48 := newNode.node48()
		newNode48.size = 0
		for i := 0; i < len(n256.children); i++ {
			if n256.children[byte(i)] == nil {
				continue
			}
			newNode48.children[newNode48.size+1] = n256.children[byte(i)]
			newNode48.keys[byte(i)] = byte(newNode48.size + 1)
			newNode48.size++
		}
		n.replaceWith(newNode)
	}
}

// longestCommonPrefix returns the longest number of bytes
// that match between the current artNode's prefix
// and the passed in artNode at the specified depth.
func (n *artNode) longestCommonPrefix(other *artNode, depth int) int {
	limit := min(len(n.leafNode().key), len(other.leafNode().key)) - depth
	for i := 0; i < limit; i++ {
		if n.leafNode().key[depth+i] != other.leafNode().key[depth+i] {
			return i
		}
	}
	return limit
}

// minSize returns the minimum number of children for the current artNode.
func (n *artNode) minSize() int {
	switch n.nodeType {
	case Node4:
		return node4Min
	case Node16:
		return node16Min
	case Node48:
		return node48Min
	case Node256:
		return node256Min
	}
	return 0
}

// maxSize returns the maximum number of children for the current artNode.
func (n *artNode) maxSize() int {
	switch n.nodeType {
	case Node4:
		return node4Max
	case Node16:
		return node16Max
	case Node48:
		return node48Max
	case Node256:
		return node256Max
	}
	return 0
}

// minimum returns the minimum child at the current artNode.
func (n *artNode) minimum() *artNode {
	if n == nil {
		return nil
	}

	switch n.nodeType {
	case LeafNode:
		return n
	case Node4:
		return n.node4().children[0].minimum()
	case Node16:
		return n.node16().children[0].minimum()
	case Node48:
		i := 0
		for n.node48().keys[i] == 0 {
			i++
		}
		return n.node48().children[n.node48().keys[i]].minimum()
	case Node256:
		i := 0
		for n.node256().children[i] == nil {
			i++
		}
		return n.node256().children[i].minimum()
	}

	return n
}

//maximum returns the maximum child at the current artNode.
func (n *artNode) maximum() *artNode {
	if n == nil {
		return nil
	}

	switch n.nodeType {
	case LeafNode:
		return n
	case Node4:
		n4 := n.node4()
		return n4.children[n4.size-1].maximum()
	case Node16:
		n16 := n.node16()
		return n16.children[n16.size-1].maximum()
	case Node48:
		n48 := n.node48()
		i := len(n48.keys) - 1
		for n48.keys[i] == 0 {
			i--
		}
		return n48.children[n48.keys[i]].maximum()
	case Node256:
		n256 := n.node256()
		i := len(n256.children) - 1
		for i > 0 && n256.children[byte(i)] == nil {
			i--
		}
		return n256.children[i].maximum()
	}
	return nil
}

// node returns the metadata node of the current artNode.
func (n *artNode) node() *node {
	return (*node)(n.nodePtr)
}

// node4 returns the metadata node4 of the current artNode.
func (n *artNode) node4() *node4 {
	return (*node4)(n.nodePtr)
}

// node16 returns the metadata node16 of the current artNode.
func (n *artNode) node16() *node16 {
	return (*node16)(n.nodePtr)
}

// node48 returns the metadata node48 of the current artNode.
func (n *artNode) node48() *node48 {
	return (*node48)(n.nodePtr)
}

// node256 returns the metadata node256 of the current artNode.
func (n *artNode) node256() *node256 {
	return (*node256)(n.nodePtr)
}

// leafNode returns the metadata leafNode of the current artNode.
func (n *artNode) leafNode() *leafNode {
	return (*leafNode)(n.nodePtr)
}

// replaceWith replaces the current artNode with the passed in artNode.
func (n *artNode) replaceWith(other *artNode) {
	*n = *other
}

// copyMeta copies the prefix and size metadata from the passed in artNode
// to the current artNode.
func (n *artNode) copyMeta(src *artNode) {
	if src == nil {
		return
	}
	to := n.node()
	from := src.node()
	to.size = from.size
	to.prefixLen = from.prefixLen

	for i, limit := 0, min(from.prefixLen, maxPrefixLen); i < limit; i++ {
		to.prefix[i] = from.prefix[i]
	}
}

// min returns the smallest of the two passed in integers.
func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
