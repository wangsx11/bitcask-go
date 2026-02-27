package index

import (
	"bitcask-go/data"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBTree_Put(t *testing.T) {
	bt := NewBTree()

	// 测试添加一个nil
	res1 := bt.Put(nil, &data.LogRecordPos{Fid: 1, Offset: 100})
	assert.Nil(t, res1)

	//测试添加一个正常的数据
	res2 := bt.Put([]byte("a"), &data.LogRecordPos{Fid: 1, Offset: 2})
	assert.Nil(t, res2)

	res3 := bt.Put([]byte("a"), &data.LogRecordPos{Fid: 4, Offset: 100})
	assert.Equal(t, uint32(1), res3.Fid)
	assert.Equal(t, int64(2), res3.Offset)
}

func TestBTree_Get(t *testing.T) {

	bt := NewBTree()
	res1 := bt.Put(nil, &data.LogRecordPos{Fid: 1, Offset: 100})
	assert.Nil(t, res1)

	// 测试获取一个nil
	pos1 := bt.Get(nil)
	assert.Equal(t, uint32(1), pos1.Fid)
	assert.Equal(t, int64(100), pos1.Offset)

	// 测试更新一个数据
	_ = bt.Put([]byte("a"), &data.LogRecordPos{Fid: 1, Offset: 2})
	res3 := bt.Put([]byte("a"), &data.LogRecordPos{Fid: 2, Offset: 3})
	assert.Equal(t, uint32(1), res3.Fid)
	assert.Equal(t, int64(2), res3.Offset)

	// 查看数据是否被更新
	pos2 := bt.Get([]byte("a"))

	// t.Log(pos2) 输出pos2
	assert.Equal(t, uint32(2), pos2.Fid)
	assert.Equal(t, int64(3), pos2.Offset)

}

func TestBTree_Delete(t *testing.T) {

	bt := NewBTree()

	res1 := bt.Put(nil, &data.LogRecordPos{Fid: 1, Offset: 100})
	assert.Nil(t, res1)

	// 删除key为nil的情况
	res2, ok1 := bt.Delete(nil)
	// t.Log(res2, ok1)
	assert.True(t, ok1)
	assert.Equal(t, uint32(1), res2.Fid)
	assert.Equal(t, int64(100), res2.Offset)

	res3 := bt.Put([]byte("a"), &data.LogRecordPos{Fid: 1, Offset: 2})
	assert.Nil(t, res3)

	// 正常删除key
	res4, ok2 := bt.Delete([]byte("a"))
	assert.True(t, ok2)
	assert.Equal(t, uint32(1), res4.Fid)
	assert.Equal(t, int64(2), res4.Offset)

	// 删除一个不存在的key
	res5, ok3 := bt.Delete([]byte("b"))

	assert.False(t, ok3)
	assert.Nil(t, res5)
}

func TestBTree_Iterator(t *testing.T) {
	bt1 := NewBTree()
	iter1 := bt1.Iterator(false)
	// 1. BTree 为空的情况
	assert.Equal(t, false, iter1.Valid())
	// 2. BTree 有数据的情况
	bt1.Put([]byte("aaa"), &data.LogRecordPos{Fid: 1, Offset: 10})
	iter2 := bt1.Iterator(false)
	assert.Equal(t, true, iter2.Valid())
	assert.NotNil(t, iter2.Key())
	assert.NotNil(t, iter2.Value())
	iter2.Next()
	assert.Equal(t, false, iter2.Valid())

	// 3. 有多条数据
	bt1.Put([]byte("bbb"), &data.LogRecordPos{Fid: 1, Offset: 10})
	bt1.Put([]byte("ccc"), &data.LogRecordPos{Fid: 1, Offset: 10})
	bt1.Put([]byte("ddd"), &data.LogRecordPos{Fid: 1, Offset: 10})

	iter3 := bt1.Iterator(false)
	for iter3.Rewind(); iter3.Valid(); iter3.Next() {
		// t.Log("key = ", string(iter3.Key()))
		assert.NotNil(t, iter3.Key())
	}

	iter4 := bt1.Iterator(true)
	for iter4.Rewind(); iter4.Valid(); iter4.Next() {
		// t.Log("key = ", string(iter4.Key()))
		assert.NotNil(t, iter4.Key())
	}

	// 4. 测试 seek
	iter5 := bt1.Iterator(false)
	for iter5.Seek([]byte("bbb")); iter5.Valid(); iter5.Next() {
		// t.Log(string(iter5.Key()))
		assert.NotNil(t, iter5.Valid())
	}

	// 5. 反向的 seek
	iter6 := bt1.Iterator(true)
	for iter6.Seek([]byte("ccc")); iter6.Valid(); iter6.Next() {
		// t.Log(string(iter6.Key()))
		assert.NotNil(t, iter6.Valid())
	}
}
