package bitcask_go

import (
	"bitcask-go/utils"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDB_WriteBatch(t *testing.T) {
	opts := DefaultOptions
	
	dir, _ := os.MkdirTemp("", "bitcask-go-batch-1")
	opts.DirPath = dir
	db, err := Open(opts)
	defer destroyDB(db)
	assert.Nil(t, err)
	assert.NotNil(t, db)

	// 写数据之后并不提交
	wb := db.NewWriteBatch(DefaultWriteBatchOptions)
	err = wb.Put(utils.GetTestKey(1), utils.RandomValue(10))
	assert.Nil(t, err)
	err = wb.Delete(utils.GetTestKey(2))
	assert.Nil(t, err)
	_, err = db.Get(utils.GetTestKey(1))
	assert.Equal(t, ErrKeyNotFound, err)

	// 正常提交数据
	err = wb.Commit()
	assert.Nil(t, err)

	val, err := db.Get(utils.GetTestKey(1))
	assert.Nil(t, err)
	assert.NotNil(t, val)

	// 删除数据
	wb2 := db.NewWriteBatch(DefaultWriteBatchOptions)

	err = wb2.Delete(utils.GetTestKey(1))
	assert.Nil(t, err)
	err = wb2.Commit()
	assert.Nil(t, err)

	val2, err := db.Get(utils.GetTestKey(1))
	assert.Equal(t, ErrKeyNotFound, err)
	assert.Nil(t, val2)
}

func TestDB_WriteBatch2(t *testing.T) {
	opts := DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-batch-2")
	opts.DirPath = dir
	db, err := Open(opts)
	defer destroyDB(db)
	assert.Nil(t, err)
	assert.NotNil(t, db)

	err = db.Put(utils.GetTestKey(1), utils.GetTestKey(10))
	assert.Nil(t, err)

	wb := db.NewWriteBatch(DefaultWriteBatchOptions)
	err = wb.Put(utils.GetTestKey(2), utils.GetTestKey(10))
	assert.Nil(t, err)

	err = wb.Delete(utils.GetTestKey(1))
	assert.Nil(t, err)

	err = wb.Commit()
	assert.Nil(t, err)

	err = wb.Put(utils.GetTestKey(11), utils.GetTestKey(11))
	assert.Nil(t, err)

	err = wb.Commit()
	assert.Nil(t, err)

	err = db.Close()
	assert.Nil(t, err)

	db2, err := Open(opts)
	assert.Nil(t, err)
	_, err = db2.Get(utils.GetTestKey(1))
	assert.Equal(t, ErrKeyNotFound, err)

	assert.Equal(t, uint64(2), db.seqNo) // 校验事务序列号 Commit了两次，所以当前序列号为2

}

// func TestDB_WriteBatch3(t *testing.T) {
// 	opts := DefaultOptions
// 	// dir, _ := os.MkdirTemp("", "bitcask-go-batch-3")
// 	dir := "/tmp/bitcask-go-batch-3"
// 	opts.DirPath = dir
// 	db, err := Open(opts)
// 	// defer destroyDB(db)
// 	assert.Nil(t, err)
// 	assert.NotNil(t, db)

// 	keys := db.ListKeys()
// 	t.Log(len(keys))

// 	wbOptions := DefaultWriteBatchOptions
// 	wbOptions.MaxBatchNum = 1000000
// 	wb := db.NewWriteBatch(wbOptions)

// 	// 如果在执行过程中手动中止程序(没有执行Commit操作， 则len(db.ListKeys()) 值为0)
// 	for i := 0; i < 500000; i++ {
// 		t.Log(i)
// 		err := wb.Put(utils.GetTestKey(i), utils.RandomValue(1024))
// 		assert.Nil(t, err)
// 	}
// 	err = wb.Commit()
// 	assert.Nil(t, err)

// 	err = db.Close()
// 	assert.Nil(t, err)
// }
