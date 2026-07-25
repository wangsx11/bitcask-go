package bitcask_go

import (
	"bitcask-go/utils"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDB_WriteBatch(t *testing.T) {
	opts := testOptions(t)
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
	opts := testOptions(t)
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
	defer destroyDB(db2)
	assert.Nil(t, err)
	_, err = db2.Get(utils.GetTestKey(1))
	assert.Equal(t, ErrKeyNotFound, err)

	assert.Equal(t, uint64(2), db.seqNo) // 校验事务序列号 Commit了两次，所以当前序列号为2

}
