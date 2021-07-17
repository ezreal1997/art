package art

import (
	"encoding/binary"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"art/testdata"
)

func TestArtTreeInsert(t *testing.T) {
	tree := newArt()
	tree.Insert(Key("hello"), "world")

	assert.Equal(t, int64(1), tree.size)
	assert.IsType(t, LeafNode, tree.root.nodeType)
}

func TestArtTreeInsertAndSearch(t *testing.T) {
	tree := newArt()

	tree.Insert(Key("hello"), "world")
	res := tree.Search(Key("hello"))

	assert.Equal(t, "world", res)
}

func TestArtTreeInsert2AndSearch(t *testing.T) {
	tree := newArt()

	tree.Insert(Key("hello"), "world")
	tree.Insert(Key("yo"), "earth")

	res := tree.Search(Key("yo"))
	assert.NotNil(t, res)
	assert.Equal(t, "earth", res)

	res2 := tree.Search([]byte("hello"))
	assert.NotNil(t, res2)
	assert.Equal(t, "world", res2)
}

func TestArtTreeInsert2WithSimilarPrefix(t *testing.T) {
	tree := newArt()

	tree.Insert(Key("a"), "a")
	tree.Insert(Key("aa"), "aa")

	res := tree.Search(Key("aa"))

	assert.NotNil(t, res)
	assert.Equal(t, "aa", res)
}

func TestArtTreeInsert3AndSearchWords(t *testing.T) {
	tree := newArt()

	searchTerms := []string{"A", "a", "aa"}

	for i := range searchTerms {
		tree.Insert(Key(searchTerms[i]), searchTerms[i])
	}

	for i := range searchTerms {
		res := tree.Search(Key(searchTerms[i]))
		assert.NotNil(t, res)
		assert.Equal(t, searchTerms[i], res)
	}
}

func TestArtTreeInsertAndGrowToBiggerNode(t *testing.T) {
	var testData = []struct {
		totalNodes byte
		expected   NodeType
	}{
		{5, Node16},
		{17, Node48},
		{49, Node256},
	}

	for _, data := range testData {
		tree := newArt()
		for i := byte(0); i < data.totalNodes; i++ {
			tree.Insert(Key{i}, i)
		}
		assert.Equal(t, int64(data.totalNodes), tree.size)
		assert.Equal(t, data.expected, tree.root.nodeType)
	}
}

func TestInsertManyWordsAndEnsureSearchResultAndMinimumMaximum(t *testing.T) {
	tree := newArt()

	words := testdata.LoadTestFile("testdata/data/words.txt")

	for _, w := range words {
		tree.Insert(w, w)
	}

	for _, w := range words {
		res := tree.Search(w)
		assert.NotNil(t, res)
		assert.Equal(t, w, res)
	}

	minimum := tree.root.minimum()
	assert.Equal(t, []byte("A"), minimum.Value().([]byte))

	maximum := tree.root.maximum()
	assert.Equal(t, []byte("zythum"), maximum.Value().([]byte))
}

func TestInsertManyUUIDsAndEnsureSearchAndMinimumMaximum(t *testing.T) {
	tree := newArt()

	uuids := testdata.LoadTestFile("testdata/data/uuid.txt")

	for _, uuid := range uuids {
		tree.Insert(uuid, uuid)
	}

	for _, uuid := range uuids {
		res := tree.Search(uuid)

		assert.NotNil(t, res)
		assert.Equal(t, res, uuid)
	}

	minimum := tree.root.minimum()
	assert.NotNil(t, minimum.Value())
	assert.Equal(t, []byte("00005076-6244-4739-808b-a58512fd6642"), minimum.Value().([]byte))

	maximum := tree.root.maximum()
	assert.NotNil(t, maximum.Value())
	assert.Equal(t, []byte("ffffb7f1-20de-4a46-a3ec-8c87d5c7fce0"), maximum.Value().([]byte))
}

func TestInsertAndRemove1(t *testing.T) {
	tree := newArt()

	tree.Insert(Key("test"), []byte("data"))

	tree.Delete(Key("test"))

	assert.Zero(t, tree.size)
	assert.Nil(t, tree.root)
}

func TestInsert2AndRemove1AndRootShouldBeLeafNode(t *testing.T) {
	tree := newArt()

	tree.Insert(Key("test"), []byte("data"))
	tree.Insert(Key("test2"), []byte("data"))

	tree.Delete(Key("test"))

	assert.Equal(t, int64(1), tree.size)
	assert.NotNil(t, tree.root)
	assert.IsType(t, LeafNode, tree.root.nodeType)
}

func TestInsert2AndRemove2AndRootShouldBeNil(t *testing.T) {
	tree := newArt()

	tree.Insert(Key("test"), []byte("data"))
	tree.Insert(Key("test2"), []byte("data"))

	tree.Delete(Key("test"))
	tree.Delete(Key("test2"))

	assert.Zero(t, tree.size)
	assert.Nil(t, tree.root)
}

func TestInsert5AndRemove1AndRootShouldBeNode4(t *testing.T) {
	tree := newArt()

	for i := 0; i < 5; i++ {
		tree.Insert(Key{byte(i)}, []byte{byte(i)})
	}

	tree.Delete(Key{1})
	res := *(tree.root.findChild(byte(1)))

	assert.Nil(t, res)
	assert.Equal(t, int64(4), tree.size)
	assert.NotNil(t, tree.root)
	assert.IsType(t, Node4, tree.root.nodeType)
}

func TestInsert5AndRemove5AndRootShouldBeNil(t *testing.T) {
	tree := newArt()

	for i := 0; i < 5; i++ {
		tree.Insert(Key{byte(i)}, []byte{byte(i)})
	}

	for i := 0; i < 5; i++ {
		tree.Delete(Key{byte(i)})
	}

	res := tree.root.findChild(byte(1))

	assert.Condition(t, func() bool {
		return res == nil || *res == nil
	})
	assert.Zero(t, tree.size)
	assert.Nil(t, tree.root)
}

func TestInsert17AndRemove1AndRootShouldBeNode16(t *testing.T) {
	tree := newArt()

	for i := 0; i < 17; i++ {
		tree.Insert(Key{byte(i)}, []byte{byte(i)})
	}

	tree.Delete(Key{2})
	res := *(tree.root.findChild(byte(2)))

	assert.Nil(t, res)
	assert.Equal(t, int64(16), tree.size)
	assert.NotNil(t, tree.root)
	assert.IsType(t, Node16, tree.root.nodeType)
}

func TestInsert17AndRemove17AndRootShouldBeNil(t *testing.T) {
	tree := newArt()

	for i := 0; i < 17; i++ {
		tree.Insert(Key{byte(i)}, []byte{byte(i)})
	}

	for i := 0; i < 17; i++ {
		tree.Delete(Key{byte(i)})
	}

	res := tree.root.findChild(byte(1))

	assert.Condition(t, func() bool {
		return res == nil || *res == nil
	})
	assert.Zero(t, tree.size)
	assert.Nil(t, tree.root)
}

func TestInsert49AndRemove1AndRootShouldBeNode48(t *testing.T) {
	tree := newArt()

	for i := 0; i < 49; i++ {
		tree.Insert(Key{byte(i)}, []byte{byte(i)})
	}

	tree.Delete(Key{2})
	res := *(tree.root.findChild(byte(2)))
	assert.Nil(t, res)

	assert.Equal(t, int64(48), tree.size)

	assert.NotNil(t, tree.root)
	assert.IsType(t, Node48, tree.root.nodeType)
}

func TestInsert49AndRemove49AndRootShouldBeNil(t *testing.T) {
	tree := newArt()

	for i := 0; i < 49; i++ {
		tree.Insert(Key{byte(i)}, []byte{byte(i)})
	}

	for i := 0; i < 49; i++ {
		tree.Delete(Key{byte(i)})
	}

	res := tree.root.findChild(byte(1))
	assert.Condition(t, func() bool {
		return res == nil || *res == nil
	})
	assert.Zero(t, tree.size)
	assert.Nil(t, tree.root)
}

func TestEachPreOrder(t *testing.T) {
	tree := newArt()
	tree.Insert(Key("1"), []byte("1"))
	tree.Insert(Key("2"), []byte("2"))

	var traversal []Node

	tree.Each(func(node Node) {
		traversal = append(traversal, node)
	})

	assert.Equal(t, traversal[0], tree.root)
	assert.Equal(t, Node4, traversal[0].NodeType())

	assert.Equal(t, traversal[1].Key(), Key("1"))
	assert.Equal(t, LeafNode, traversal[1].NodeType())

	assert.Equal(t, traversal[2].Key(), Key("2"))
	assert.Equal(t, LeafNode, traversal[2].NodeType())
}

func TestEachNode48(t *testing.T) {
	tree := newArt()

	for i := 48; i > 0; i-- {
		tree.Insert(Key{byte(i)}, []byte{byte(i)})
	}

	var traversal []Node

	tree.Each(func(node Node) {
		traversal = append(traversal, node)
	})

	assert.Equal(t, traversal[0], tree.root)
	assert.Equal(t, Node48, traversal[0].NodeType())

	for i := 1; i < 48; i++ {
		assert.Equal(t, traversal[i].Key(), Key{byte(i)})
		assert.Equal(t, LeafNode, traversal[i].NodeType())
	}
}

func TestEachFullIterationExpectCountOfAllTypes(t *testing.T) {
	tree := newArt()

	words := testdata.LoadTestFile("testdata/data/words.txt")

	for _, w := range words {
		tree.Insert(w, w)
	}

	var leafCount = 0
	var node4Count = 0
	var node16Count = 0
	var node48Count = 0
	var node256Count = 0

	tree.Each(func(node Node) {
		switch node.NodeType() {
		case Node4:
			node4Count++
		case Node16:
			node16Count++
		case Node48:
			node48Count++
		case Node256:
			node256Count++
		case LeafNode:
			leafCount++
		default:
		}
	})

	assert.Equalf(t, 235886, leafCount, "leafNode count must be equal to 235886")
	assert.Equalf(t, 111616, node4Count, "node4 count must be equal to 111616")
	assert.Equalf(t, 12181, node16Count, "node16 count must be equal to 12181")
	assert.Equalf(t, 458, node48Count, "node48 count must be equal to 458")
	assert.Equalf(t, 1, node256Count, "node256 must be the only one")
}

func TestInsertManyWordsAndRemoveThemAll(t *testing.T) {
	tree := newArt()

	words := testdata.LoadTestFile("testdata/data/words.txt")

	for _, w := range words {
		tree.Insert(w, w)
	}

	numFound := 0

	for _, w := range words {
		tree.Delete(w)
		dblCheck := tree.Search(w)
		if dblCheck != nil {
			numFound++
		}
	}

	assert.Zero(t, tree.size)
	assert.Nil(t, tree.root)
}

func TestInsertManyUUIDsAndRemoveThemAll(t *testing.T) {
	tree := newArt()

	uuids := testdata.LoadTestFile("testdata/data/uuid.txt")

	for _, uuid := range uuids {
		tree.Insert(uuid, uuid)
	}

	numFound := 0

	for _, uuid := range uuids {
		tree.Delete(uuid)

		dblCheck := tree.Search(uuid)
		if dblCheck != nil {
			numFound++
		}
	}
	assert.Zero(t, tree.size)
	assert.Nil(t, tree.root)
}

func TestInsertWithSameByteSliceAddress(t *testing.T) {
	rand.Seed(42)
	key := make([]byte, 8)
	tree := newArt()

	keys := make(map[string]bool)

	for i := 0; i < 135; i++ {
		binary.BigEndian.PutUint64(key, uint64(rand.Int63()))
		tree.Insert(key, key)
		keys[string(key)] = true
	}

	assert.Equal(t, int64(len(keys)), tree.size)

	for k := range keys {
		n := tree.Search(Key(k))
		assert.NotNil(t, n)
	}
}

func BenchmarkWordsTreeInsert(b *testing.B) {
	words := testdata.LoadTestFile("testdata/data/words.txt")
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		tree := newArt()
		for _, w := range words {
			tree.Insert(w, w)
		}
	}
}

func BenchmarkWordsTreeSearch(b *testing.B) {
	words := testdata.LoadTestFile("testdata/data/words.txt")
	tree := newArt()
	for _, w := range words {
		tree.Insert(w, w)
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for _, w := range words {
			tree.Search(w)
		}
	}
}

func BenchmarkWordsTreeForEach(b *testing.B) {
	words := testdata.LoadTestFile("testdata/data/words.txt")
	tree := newArt()
	for _, w := range words {
		tree.Insert(w, w)
	}
	b.ResetTimer()

	nodeTypes := make(map[NodeType]int)
	tree.Each(func(n Node) {
		nodeTypes[n.NodeType()]++
	})
	assert.Equal(b, map[NodeType]int{LeafNode: 235886, Node4: 111616, Node16: 12181, Node48: 458, Node256: 1}, nodeTypes)
}

func BenchmarkUUIDsTreeInsert(b *testing.B) {
	words := testdata.LoadTestFile("testdata/data/uuid.txt")
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		tree := newArt()
		for _, w := range words {
			tree.Insert(w, w)
		}
	}
}

func BenchmarkUUIDsTreeSearch(b *testing.B) {
	words := testdata.LoadTestFile("testdata/data/uuid.txt")
	tree := newArt()
	for _, w := range words {
		tree.Insert(w, w)
	}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for _, w := range words {
			tree.Search(w)
		}
	}
}

func BenchmarkUUIDsTreeEach(b *testing.B) {
	words := testdata.LoadTestFile("testdata/data/uuid.txt")
	tree := newArt()
	for _, w := range words {
		tree.Insert(w, w)
	}
	b.ResetTimer()

	nodeTypes := make(map[NodeType]int)
	tree.Each(func(n Node) {
		nodeTypes[n.NodeType()]++
	})
	assert.Equal(b, map[NodeType]int{LeafNode: 500000, Node4: 103602, Node16: 56030}, nodeTypes)
}
