package index

import (
	"bitcask-go/data"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBPlusTree_Put(t *testing.T) {
	path := filepath.Join(os.TempDir(), "bptree-put")
	os.MkdirAll(path, os.ModePerm)
	defer func() {
		_ = os.RemoveAll(path)
	}()
	tree := NewBPlusTree(path, false)

	res1 := tree.Put([]byte("key-1"), &data.LogRecordPos{Fid: 1, Offset: 100})
	assert.Nil(t, res1)
	res2 := tree.Put([]byte("key-2"), &data.LogRecordPos{Fid: 2, Offset: 100})
	assert.Nil(t, res2)

	res3 := tree.Put([]byte("key-3"), &data.LogRecordPos{Fid: 3, Offset: 100})
	assert.Nil(t, res3)

	res4 := tree.Put([]byte("key-1"), &data.LogRecordPos{Fid: 4, Offset: 200})
	assert.Equal(t, uint32(1), res4.Fid)
	assert.Equal(t, int64(100), res4.Offset)

}

func TestBPlusTree_Get(t *testing.T) {
	path := filepath.Join(os.TempDir(), "bptree-get")
	os.MkdirAll(path, os.ModePerm)
	tree := NewBPlusTree(path, false)
	defer func() {
		_ = os.RemoveAll(path)
	}()
	pos := tree.Get([]byte("not exists"))
	assert.Nil(t, pos)

	tree.Put([]byte("key"), &data.LogRecordPos{Fid: 1, Offset: 100})
	pos = tree.Get([]byte("key"))
	assert.NotNil(t, pos)

	tree.Put([]byte("key"), &data.LogRecordPos{Fid: 2, Offset: 100})
	pos = tree.Get([]byte("key"))
	assert.NotNil(t, pos)
}

func TestBPlusTree_Delete(t *testing.T) {
	path := filepath.Join(os.TempDir(), "bptree-delete")
	_ = os.MkdirAll(path, os.ModePerm)
	defer func() {
		_ = os.RemoveAll(path)
	}()
	err := os.MkdirAll(filepath.Dir(path), 0755)
	assert.NoError(t, err)

	tree := NewBPlusTree(path, false)
	res1, ok1 := tree.Delete([]byte("not exists"))
	// t.Log(res1)
	assert.False(t, ok1)
	assert.Nil(t, res1)


	tree.Put([]byte("key"), &data.LogRecordPos{Fid: 1, Offset: 100})
	res2, ok2 := tree.Delete([]byte("key"))
	assert.True(t, ok2)
	assert.Equal(t, uint32(1), res2.Fid)
	assert.Equal(t, int64(100), res2.Offset)

	pos1 := tree.Get([]byte("key"))
	assert.Nil(t, pos1)
}

func TestBPlusTree_Size(t *testing.T) {
	path := filepath.Join(os.TempDir(), "bptree-size")
	_ = os.MkdirAll(path, os.ModePerm)
	defer func() {
		_ = os.RemoveAll(path)
	}()
	err := os.MkdirAll(filepath.Dir(path), 0755)
	assert.NoError(t, err)

	tree := NewBPlusTree(path, false)

	

	tree.Put([]byte("key"), &data.LogRecordPos{Fid: 1, Offset: 100})
	tree.Put([]byte("key-2"), &data.LogRecordPos{Fid: 2, Offset: 100})
	tree.Put([]byte("key-3"), &data.LogRecordPos{Fid: 3, Offset: 100})

	assert.Equal(t, 3, tree.Size())
	tree.Delete([]byte("key"))
	assert.Equal(t, 2, tree.Size())
}

func TestBPlusTree_Iterator(t *testing.T) {
	path := filepath.Join(os.TempDir(), "bptree-iterator")
	_ = os.MkdirAll(path, os.ModePerm)
	defer func() {
		_ = os.RemoveAll(path)
	}()
	tree := NewBPlusTree(path, false)

	tree.Put([]byte("caac"), &data.LogRecordPos{Fid: 1, Offset: 100})
	tree.Put([]byte("bbca"), &data.LogRecordPos{Fid: 2, Offset: 100})
	tree.Put([]byte("acce"), &data.LogRecordPos{Fid: 3, Offset: 100})
	tree.Put([]byte("bbba"), &data.LogRecordPos{Fid: 4, Offset: 100})

	it := tree.Iterator(true)
	for it.Rewind(); it.Valid(); it.Next() {
		// t.Log(string(it.Key()))
		assert.NotNil(t, it.Key())
		assert.NotNil(t, it.Value())
	}
}
