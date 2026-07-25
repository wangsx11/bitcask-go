package index

import (
	"bitcask-go/data"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdaptiveRadixTree_Put(t *testing.T) {
	art := NewART()
	res1 := art.Put([]byte("a"), &data.LogRecordPos{Fid: 1, Offset: 100})
	assert.Nil(t, res1)

	res2 := art.Put([]byte("a"), &data.LogRecordPos{Fid: 4, Offset: 400})
	assert.Equal(t, uint32(1), res2.Fid)
	assert.Equal(t, int64(100), res2.Offset)

	art.Put([]byte("b"), &data.LogRecordPos{Fid: 2, Offset: 200})

	art.Put([]byte("c"), &data.LogRecordPos{Fid: 3, Offset: 300})

}

func TestAdaptiveRadixTree_Get(t *testing.T) {
	art := NewART()
	res1 := art.Put([]byte("a"), &data.LogRecordPos{Fid: 1, Offset: 100})
	assert.Nil(t, res1)
	pos := art.Get([]byte("a"))
	assert.NotNil(t, pos)
	pos1 := art.Get([]byte("b"))
	assert.Nil(t, pos1)

	res2 := art.Put([]byte("a"), &data.LogRecordPos{Fid: 2, Offset: 200})
	pos2 := art.Get([]byte("a"))
	assert.Equal(t, uint32(1), res2.Fid)
	assert.Equal(t, int64(100), res2.Offset)

	assert.NotNil(t, pos2)

}

func TestAdaptiveRadixTree_Delete(t *testing.T) {
	art := NewART()
	res1, ok1 := art.Delete([]byte("nit exists"))
	assert.False(t, ok1)
	assert.Nil(t, res1)
	art.Put([]byte("a"), &data.LogRecordPos{Fid: 1, Offset: 100})
	res2, ok2 := art.Delete([]byte("a"))
	assert.True(t, ok2)
	assert.Equal(t, uint32(1), res2.Fid)
	assert.Equal(t, int64(100), res2.Offset)

	pos := art.Get([]byte("a"))
	assert.Nil(t, pos)
}

func TestAdaptiveRadixTree_Size(t *testing.T) {
	art := NewART()
	size := art.Size()
	assert.Equal(t, 0, size)
	art.Put([]byte("a"), &data.LogRecordPos{Fid: 1, Offset: 100})
	art.Put([]byte("b"), &data.LogRecordPos{Fid: 2, Offset: 200})
	art.Put([]byte("c"), &data.LogRecordPos{Fid: 3, Offset: 300})
	art.Put([]byte("c"), &data.LogRecordPos{Fid: 4, Offset: 300})
	size = art.Size()
	assert.Equal(t, 3, size)
	art.Delete([]byte("a"))
	size = art.Size()
	assert.Equal(t, 2, size)
}

func TestAdaptiveRadixTree_Iterator(t *testing.T) {
	art := NewART()
	emptyIterator := art.Iterator(false)
	emptyIterator.Close()

	art.Put([]byte("ccde"), &data.LogRecordPos{Fid: 1, Offset: 100})
	art.Put([]byte("adse"), &data.LogRecordPos{Fid: 2, Offset: 200})
	art.Put([]byte("bbde"), &data.LogRecordPos{Fid: 3, Offset: 300})
	art.Put([]byte("bade"), &data.LogRecordPos{Fid: 4, Offset: 400})

	iter := art.Iterator(true)
	defer iter.Close()
	for iter.Rewind(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()
		// t.Log(string(key))
		assert.NotNil(t, key)
		assert.NotNil(t, value)
	}
}
