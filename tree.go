package art

// tree - adaptive radix tree type.
type tree struct {
	root *artNode
	size int64
}

// newArt returns art with 0 nodes.
func newArt() *tree {
	return &tree{root: nil, size: 0}
}

// Search returns the node that contains the passed in key, or nil if not found.
func (t *tree) Search(key Key) Value {
	return t.searchHelper(t.root, key, 0)
}

// searchHelper is a helper function for Search.
func (t *tree) searchHelper(current *artNode, key []byte, depth int) interface{} {
	for current != nil {
		if current.isLeaf() {
			if current.isMatch(key) {
				return current.leafNode().value
			}
			return nil
		}
		if current.prefixMismatch(key, depth) != current.node().prefixLen {
			return nil
		}
		depth += current.node().prefixLen

		var keyChar byte
		if depth < 0 || depth >= len(key) {
			keyChar = byte(0)
		} else {
			keyChar = key[depth]
		}
		current = *(current.findChild(keyChar))
		depth++
	}

	return nil
}

// Insert inserts the passed in value that is indexed by the passed in key into the tree.
func (t *tree) Insert(key Key, value Value) {
	t.insertHelper(&t.root, key, value, 0)
}

// insertHelper is a helper function for Insert.
func (t *tree) insertHelper(currentRef **artNode, key []byte, value interface{}, depth int) {
	if *currentRef == nil {
		*currentRef = newLeafNode(key, value)
		t.size++
		return
	}
	current := *currentRef

	if current.isLeaf() {
		// NOTE: Currently, overwrite if the key matches.
		if current.isMatch(key) {
			current.leafNode().value = value
			return
		}

		newNode4 := newNode4()
		newLeafNode := newLeafNode(key, value)

		limit := current.longestCommonPrefix(newLeafNode, depth)

		newNode4.node().prefixLen = limit

		memcpy(newNode4.node().prefix[:], key[depth:], min(newNode4.node().prefixLen, maxPrefixLen))

		if depth+newNode4.node().prefixLen < 0 || depth+newNode4.node().prefixLen >= len(current.leafNode().key) {
			newNode4.addChild(0, current)
		} else {
			newNode4.addChild(current.leafNode().key[depth+newNode4.node().prefixLen], current)
		}

		if depth+newNode4.node().prefixLen < 0 || depth+newNode4.node().prefixLen >= len(key) {
			newNode4.addChild(0, newLeafNode)
		} else {
			newNode4.addChild(key[depth+newNode4.node().prefixLen], newLeafNode)
		}

		*currentRef = newNode4
		t.size++

		return
	}

	node := current.node()
	if node.prefixLen != 0 {
		mismatch := current.prefixMismatch(key, depth)
		if mismatch != node.prefixLen {
			newNode4 := newNode4()
			*currentRef = newNode4
			newNode4.node().prefixLen = mismatch

			memcpy(newNode4.node().prefix[:], node.prefix[:], mismatch)

			if node.prefixLen < maxPrefixLen {
				newNode4.addChild(node.prefix[mismatch], current)
				node.prefixLen -= mismatch + 1
				memmove(node.prefix[:], node.prefix[mismatch+1:], min(node.prefixLen, maxPrefixLen))
			} else {
				node.prefixLen -= mismatch + 1
				minKey := current.minimum().leafNode().key
				newNode4.addChild(minKey[depth+mismatch], current)
				memmove(node.prefix[:], minKey[depth+mismatch+1:], min(node.prefixLen, maxPrefixLen))
			}

			newLeafNode := newLeafNode(key, value)
			newNode4.addChild(key[depth+mismatch], newLeafNode)

			t.size++
			return
		}
		depth += node.prefixLen
	}

	next := current.findChild(key[depth])
	if *next != nil {
		t.insertHelper(next, key, value, depth+1)
	} else {
		current.addChild(key[depth], newLeafNode(key, value))
		t.size++
	}
}

// Delete deletes the child of the passed in key.
func (t *tree) Delete(key []byte) bool {
	return t.deleteHelper(&t.root, key, 0)
}

// deleteHelper is a helper function of Delete.
func (t *tree) deleteHelper(currentRef **artNode, key []byte, depth int) bool {
	if t == nil || *currentRef == nil || len(key) == 0 {
		return false
	}

	current := *currentRef
	if current.isLeaf() {
		if current.isMatch(key) {
			*currentRef = nil
			t.size--
			return true
		}
	}

	if current.node().prefixLen != 0 {
		mismatch := current.prefixMismatch(key, depth)
		if mismatch != current.node().prefixLen {
			return false
		}
		depth += current.node().prefixLen
	}

	var keyChar byte
	if depth < 0 || depth >= len(key) {
		keyChar = byte(0)
	} else {
		keyChar = key[depth]
	}
	next := current.findChild(keyChar)

	if *next != nil && (*next).isLeaf() && (*next).isMatch(key) {
		current.RemoveChild(keyChar)
		t.size--
		return true
	}

	return t.deleteHelper(next, key, depth+1)
}

// Each iterate the whole tree with the lexicographical order,
// and will call the given callback for each tree node.
func (t *tree) Each(callback Callback) {
	t.eachHelper(t.root, callback)
}

// Size returns the number of leafNodes (key-value) in the tree.
func (t *tree) Size() int {
	return int(t.size)
}

// eachHelper is a helper function of Each.
func (t *tree) eachHelper(current *artNode, callback Callback) {
	if current == nil {
		return
	}

	callback(current)

	switch current.nodeType {
	case Node4:
		t.eachChildren(current.node4().children[:], callback)
	case Node16:
		t.eachChildren(current.node16().children[:], callback)
	case Node48:
		node := current.node48()
		for _, i := range node.keys {
			if i > 0 {
				next := current.node48().children[i]
				if next != nil {
					t.eachHelper(next, callback)
				}
			}
		}
	case Node256:
		t.eachChildren(current.node256().children[:], callback)
	}
}

// eachChildren is used by eachHelper to iterate children of artNode.
func (t *tree) eachChildren(children []*artNode, callback Callback) {
	for _, child := range children {
		if child != nil {
			t.eachHelper(child, callback)
		}
	}
}

// memcpy copies numBytes bytes from src to dst.
func memcpy(dst []byte, src []byte, numBytes int) {
	for i := 0; i < numBytes && i < len(src) && i < len(dst); i++ {
		dst[i] = src[i]
	}
}

// memmove moves numBytes bytes from src to dst.
func memmove(dst []byte, src []byte, numBytes int) {
	for i := 0; i < numBytes; i++ {
		dst[i] = src[i]
	}
}
