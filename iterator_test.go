package bitcask_go

import (
	"bitcask-go/utils"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDB_NewIterator(t *testing.T) {
	opts := testOptions(t)
	opts.DataFileSize = 64 * 1024 * 1024
	db, err := Open(opts)
	defer destroyDB(db)
	assert.Nil(t, err)
	assert.NotNil(t, db)

	iterator := db.NewIterator(DefaultIteratorOptions)
	defer iterator.Close()
	assert.NotNil(t, iterator)
	assert.Equal(t, false, iterator.Valid())
	t.Log(iterator.Valid())
}

func TestDB_NewIterator_One_Value(t *testing.T) {
	opts := testOptions(t)
	opts.DataFileSize = 64 * 1024 * 1024
	db, err := Open(opts)
	defer destroyDB(db)
	assert.Nil(t, err)
	assert.NotNil(t, db)

	err = db.Put(utils.GetTestKey(10), utils.GetTestKey(10))
	assert.Nil(t, err)

	iterator := db.NewIterator(DefaultIteratorOptions)
	defer iterator.Close()
	assert.NotNil(t, iterator)
	assert.Equal(t, true, iterator.Valid())
	t.Log(iterator.Valid())
	t.Log(string(iterator.Key()))
	assert.Equal(t, utils.GetTestKey(10), iterator.Key())
	val, err := iterator.Value()
	assert.Nil(t, err)
	assert.Equal(t, utils.GetTestKey(10), val)
}

func TestDB_NewIterator_Mul_Values(t *testing.T) {
	opts := testOptions(t)
	opts.DataFileSize = 64 * 1024 * 1024
	db, err := Open(opts)
	defer destroyDB(db)
	assert.Nil(t, err)
	assert.NotNil(t, db)

	// err = db.Put(utils.GetTestKey(10), utils.GetTestKey(10))
	err = db.Put([]byte("name"), []byte("wsx"))
	assert.Nil(t, err)
	err = db.Put([]byte("age"), []byte("22"))
	assert.Nil(t, err)
	err = db.Put([]byte("sex"), []byte("male"))
	assert.Nil(t, err)
	err = db.Put([]byte("addr"), []byte("china"))
	assert.Nil(t, err)
	assert.Nil(t, err)

	// 正向迭代
	iter1 := db.NewIterator(DefaultIteratorOptions)
	defer iter1.Close()
	for iter1.Rewind(); iter1.Valid(); iter1.Next() {
		t.Log("key = ", string(iter1.Key()))
		assert.NotNil(t, iter1.Key())
	}
	iter1.Rewind()
	for iter1.Seek([]byte("name")); iter1.Valid(); iter1.Next() {
		t.Log("key = ", string(iter1.Key()))
		assert.NotNil(t, iter1.Key())
	}

	// 反向迭代
	iter_opts1 := DefaultIteratorOptions
	iter_opts1.Reverse = true
	iter2 := db.NewIterator(iter_opts1)
	defer iter2.Close()
	for iter2.Rewind(); iter2.Valid(); iter2.Next() {
		t.Log("key = ", string(iter2.Key()))
		assert.NotNil(t, iter2.Key())
	}

	iter2.Rewind()
	for iter2.Seek([]byte("name")); iter2.Valid(); iter2.Next() {
		t.Log("key = ", string(iter2.Key()))
		assert.NotNil(t, iter2.Key())
	}

	// 指定 prefix
	iter_opts2 := DefaultIteratorOptions
	iter_opts2.Prefix = []byte("a")
	iter3 := db.NewIterator(iter_opts2)
	defer iter3.Close()
	for iter3.Rewind(); iter3.Valid(); iter3.Next() {
		t.Log("key = ", string(iter3.Key()))
		assert.NotNil(t, iter3.Key())
	}

}

func TestIterator(t *testing.T) {
	opts := testOptions(t)
	opts.DataFileSize = 1 * 1024 * 1024
	db, err := Open(opts)
	assert.Nil(t, err)
	assert.NotNil(t, db)
	defer destroyDB(db)
	db.Put([]byte("aaa"), []byte("value1"))
	// db.Put([]byte("bbb"), []byte("value2"))
	// db.Put([]byte("ccc"), []byte("value3"))
	// db.Put([]byte("ddd"), []byte("value4"))
	db.Put([]byte("dsdad"), []byte("value5"))
	// db.Put([]byte("dabcd"), []byte("value8"))
	// db.Put([]byte("dssdddwswd"), []byte("value6"))

	val, err := db.Get([]byte("dsdad"))
	t.Log(string(val), err)

}
