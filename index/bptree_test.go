package index

import (
	"bitcask-go/data"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newTestBPlusTree(t *testing.T) *BPlusTree {
	t.Helper()
	tree := NewBPlusTree(t.TempDir(), false)
	t.Cleanup(func() { assert.NoError(t, tree.Close()) })
	return tree
}

func TestBPlusTreePut(t *testing.T) {
	tree := newTestBPlusTree(t)

	assert.Nil(t, tree.Put([]byte("key-1"), &data.LogRecordPos{Fid: 1, Offset: 100}))
	assert.Nil(t, tree.Put([]byte("key-2"), &data.LogRecordPos{Fid: 2, Offset: 100}))
	assert.Nil(t, tree.Put([]byte("key-3"), &data.LogRecordPos{Fid: 3, Offset: 100}))

	old := tree.Put([]byte("key-1"), &data.LogRecordPos{Fid: 4, Offset: 200})
	assert.Equal(t, uint32(1), old.Fid)
	assert.Equal(t, int64(100), old.Offset)
}

func TestBPlusTreeGet(t *testing.T) {
	tree := newTestBPlusTree(t)
	assert.Nil(t, tree.Get([]byte("not-exists")))

	tree.Put([]byte("key"), &data.LogRecordPos{Fid: 1, Offset: 100})
	assert.NotNil(t, tree.Get([]byte("key")))
	tree.Put([]byte("key"), &data.LogRecordPos{Fid: 2, Offset: 100})
	assert.Equal(t, uint32(2), tree.Get([]byte("key")).Fid)
}

func TestBPlusTreeDelete(t *testing.T) {
	tree := newTestBPlusTree(t)
	old, ok := tree.Delete([]byte("not-exists"))
	assert.False(t, ok)
	assert.Nil(t, old)

	tree.Put([]byte("key"), &data.LogRecordPos{Fid: 1, Offset: 100})
	old, ok = tree.Delete([]byte("key"))
	assert.True(t, ok)
	assert.Equal(t, uint32(1), old.Fid)
	assert.Nil(t, tree.Get([]byte("key")))
}

func TestBPlusTreeSize(t *testing.T) {
	tree := newTestBPlusTree(t)
	for i, key := range []string{"key-1", "key-2", "key-3"} {
		tree.Put([]byte(key), &data.LogRecordPos{Fid: uint32(i + 1)})
	}
	assert.Equal(t, 3, tree.Size())
	tree.Delete([]byte("key-1"))
	assert.Equal(t, 2, tree.Size())
}

func TestBPlusTreeIterator(t *testing.T) {
	tree := newTestBPlusTree(t)
	for i, key := range []string{"caac", "bbca", "acce", "bbba"} {
		tree.Put([]byte(key), &data.LogRecordPos{Fid: uint32(i + 1), Offset: 100})
	}

	iterator := tree.Iterator(true)
	defer iterator.Close()
	for iterator.Rewind(); iterator.Valid(); iterator.Next() {
		assert.NotNil(t, iterator.Key())
		assert.NotNil(t, iterator.Value())
	}
}
