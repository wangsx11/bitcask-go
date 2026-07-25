package index

import (
	"bitcask-go/data"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newTestBPlusTree(t *testing.T) *BPlusTree {
	t.Helper()
	tree, err := NewBPlusTree(t.TempDir(), false)
	assert.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, tree.Close()) })
	return tree
}

func TestBPlusTreePut(t *testing.T) {
	tree := newTestBPlusTree(t)

	old, err := tree.Put([]byte("key-1"), &data.LogRecordPos{Fid: 1, Offset: 100})
	assert.NoError(t, err)
	assert.Nil(t, old)
	old, err = tree.Put([]byte("key-2"), &data.LogRecordPos{Fid: 2, Offset: 100})
	assert.NoError(t, err)
	assert.Nil(t, old)
	old, err = tree.Put([]byte("key-3"), &data.LogRecordPos{Fid: 3, Offset: 100})
	assert.NoError(t, err)
	assert.Nil(t, old)

	old, err = tree.Put([]byte("key-1"), &data.LogRecordPos{Fid: 4, Offset: 200})
	assert.NoError(t, err)
	assert.Equal(t, uint32(1), old.Fid)
	assert.Equal(t, int64(100), old.Offset)
}

func TestBPlusTreeGet(t *testing.T) {
	tree := newTestBPlusTree(t)
	pos, err := tree.Get([]byte("not-exists"))
	assert.NoError(t, err)
	assert.Nil(t, pos)

	_, _ = tree.Put([]byte("key"), &data.LogRecordPos{Fid: 1, Offset: 100})
	pos, err = tree.Get([]byte("key"))
	assert.NoError(t, err)
	assert.NotNil(t, pos)
	_, _ = tree.Put([]byte("key"), &data.LogRecordPos{Fid: 2, Offset: 100})
	pos, err = tree.Get([]byte("key"))
	assert.NoError(t, err)
	assert.Equal(t, uint32(2), pos.Fid)
}

func TestBPlusTreeDelete(t *testing.T) {
	tree := newTestBPlusTree(t)
	old, ok, err := tree.Delete([]byte("not-exists"))
	assert.NoError(t, err)
	assert.False(t, ok)
	assert.Nil(t, old)

	_, _ = tree.Put([]byte("key"), &data.LogRecordPos{Fid: 1, Offset: 100})
	old, ok, err = tree.Delete([]byte("key"))
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, uint32(1), old.Fid)
	pos, err := tree.Get([]byte("key"))
	assert.NoError(t, err)
	assert.Nil(t, pos)
}

func TestBPlusTreeSize(t *testing.T) {
	tree := newTestBPlusTree(t)
	for i, key := range []string{"key-1", "key-2", "key-3"} {
		_, _ = tree.Put([]byte(key), &data.LogRecordPos{Fid: uint32(i + 1)})
	}
	size, err := tree.Size()
	assert.NoError(t, err)
	assert.Equal(t, 3, size)
	_, _, _ = tree.Delete([]byte("key-1"))
	size, err = tree.Size()
	assert.NoError(t, err)
	assert.Equal(t, 2, size)
}

func TestBPlusTreeIterator(t *testing.T) {
	tree := newTestBPlusTree(t)
	for i, key := range []string{"caac", "bbca", "acce", "bbba"} {
		_, _ = tree.Put([]byte(key), &data.LogRecordPos{Fid: uint32(i + 1), Offset: 100})
	}

	iterator, err := tree.Iterator(true)
	assert.NoError(t, err)
	defer func() { assert.NoError(t, iterator.Close()) }()
	for iterator.Rewind(); iterator.Valid(); iterator.Next() {
		assert.NotNil(t, iterator.Key())
		assert.NotNil(t, iterator.Value())
	}
}

func TestBPlusTreeOperationsReturnErrorsAfterClose(t *testing.T) {
	tree, err := NewBPlusTree(t.TempDir(), false)
	assert.NoError(t, err)
	assert.NoError(t, tree.Close())

	_, err = tree.Put([]byte("key"), &data.LogRecordPos{})
	assert.Error(t, err)
	_, err = tree.Get([]byte("key"))
	assert.Error(t, err)
	_, _, err = tree.Delete([]byte("key"))
	assert.Error(t, err)
	_, err = tree.Size()
	assert.Error(t, err)
	_, err = tree.Iterator(false)
	assert.Error(t, err)
}

func TestNewIndexerReturnsUnsupportedError(t *testing.T) {
	idx, err := NewIndexer(99, t.TempDir(), false)
	assert.Nil(t, idx)
	assert.ErrorIs(t, err, ErrUnsupportedIndexType)
}
