package index

import (
	"bitcask-go/data"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdaptiveRadixTree_Put(t *testing.T) {
	art := NewART()
	res1, err := art.Put([]byte("a"), &data.LogRecordPos{Fid: 1, Offset: 100})
	assert.NoError(t, err)
	assert.Nil(t, res1)

	res2, err := art.Put([]byte("a"), &data.LogRecordPos{Fid: 4, Offset: 400})
	assert.NoError(t, err)
	assert.Equal(t, uint32(1), res2.Fid)
	assert.Equal(t, int64(100), res2.Offset)

	_, _ = art.Put([]byte("b"), &data.LogRecordPos{Fid: 2, Offset: 200})

	_, _ = art.Put([]byte("c"), &data.LogRecordPos{Fid: 3, Offset: 300})

}

func TestAdaptiveRadixTree_Get(t *testing.T) {
	art := NewART()
	res1, err := art.Put([]byte("a"), &data.LogRecordPos{Fid: 1, Offset: 100})
	assert.NoError(t, err)
	assert.Nil(t, res1)
	pos, err := art.Get([]byte("a"))
	assert.NoError(t, err)
	assert.NotNil(t, pos)
	pos1, err := art.Get([]byte("b"))
	assert.NoError(t, err)
	assert.Nil(t, pos1)

	res2, err := art.Put([]byte("a"), &data.LogRecordPos{Fid: 2, Offset: 200})
	assert.NoError(t, err)
	pos2, err := art.Get([]byte("a"))
	assert.NoError(t, err)
	assert.Equal(t, uint32(1), res2.Fid)
	assert.Equal(t, int64(100), res2.Offset)

	assert.NotNil(t, pos2)

}

func TestAdaptiveRadixTree_Delete(t *testing.T) {
	art := NewART()
	res1, ok1, err := art.Delete([]byte("nit exists"))
	assert.NoError(t, err)
	assert.False(t, ok1)
	assert.Nil(t, res1)
	_, _ = art.Put([]byte("a"), &data.LogRecordPos{Fid: 1, Offset: 100})
	res2, ok2, err := art.Delete([]byte("a"))
	assert.NoError(t, err)
	assert.True(t, ok2)
	assert.Equal(t, uint32(1), res2.Fid)
	assert.Equal(t, int64(100), res2.Offset)

	pos, err := art.Get([]byte("a"))
	assert.NoError(t, err)
	assert.Nil(t, pos)
}

func TestAdaptiveRadixTree_Size(t *testing.T) {
	art := NewART()
	size, err := art.Size()
	assert.NoError(t, err)
	assert.Equal(t, 0, size)
	_, _ = art.Put([]byte("a"), &data.LogRecordPos{Fid: 1, Offset: 100})
	_, _ = art.Put([]byte("b"), &data.LogRecordPos{Fid: 2, Offset: 200})
	_, _ = art.Put([]byte("c"), &data.LogRecordPos{Fid: 3, Offset: 300})
	_, _ = art.Put([]byte("c"), &data.LogRecordPos{Fid: 4, Offset: 300})
	size, err = art.Size()
	assert.NoError(t, err)
	assert.Equal(t, 3, size)
	_, _, _ = art.Delete([]byte("a"))
	size, err = art.Size()
	assert.NoError(t, err)
	assert.Equal(t, 2, size)
}

func TestAdaptiveRadixTree_Iterator(t *testing.T) {
	art := NewART()
	emptyIterator, err := art.Iterator(false)
	assert.NoError(t, err)
	assert.NoError(t, emptyIterator.Close())

	_, _ = art.Put([]byte("ccde"), &data.LogRecordPos{Fid: 1, Offset: 100})
	_, _ = art.Put([]byte("adse"), &data.LogRecordPos{Fid: 2, Offset: 200})
	_, _ = art.Put([]byte("bbde"), &data.LogRecordPos{Fid: 3, Offset: 300})
	_, _ = art.Put([]byte("bade"), &data.LogRecordPos{Fid: 4, Offset: 400})

	iter, err := art.Iterator(true)
	assert.NoError(t, err)
	defer func() { assert.NoError(t, iter.Close()) }()
	for iter.Rewind(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()
		// t.Log(string(key))
		assert.NotNil(t, key)
		assert.NotNil(t, value)
	}
}
