package data

import (
	"bitcask-go/fio"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpenDataFile(t *testing.T) {
	dataFile1, err := OpenDataFile(os.TempDir(), 0, fio.StanderFIO)
	assert.Nil(t, err)
	assert.NotNil(t, dataFile1)

	dataFile2, err := OpenDataFile(os.TempDir(), 111, fio.StanderFIO)
	assert.Nil(t, err)
	assert.NotNil(t, dataFile2)

	dataFile3, err := OpenDataFile(os.TempDir(), 111, fio.StanderFIO)
	assert.Nil(t, err)
	assert.NotNil(t, dataFile3)

	t.Log(os.TempDir())

}

func TestDataFile_Write(t *testing.T) {
	dataFile, err := OpenDataFile(os.TempDir(), 0, fio.StanderFIO)

	assert.Nil(t, err)
	assert.NotNil(t, dataFile)

	err = dataFile.Write([]byte("hello\n"))
	assert.Nil(t, err)

	err = dataFile.Write([]byte("go\n"))
	assert.Nil(t, err)

	err = dataFile.Write([]byte("nice to meet you\n"))
	assert.Nil(t, err)
	err = dataFile.Write([]byte("nice to meet you too\n"))
	assert.Nil(t, err)

}

func TestDataFile_Colse(t *testing.T) {
	dataFile, err := OpenDataFile(os.TempDir(), 0, fio.StanderFIO)

	assert.Nil(t, err)
	assert.NotNil(t, dataFile)

	err = dataFile.Write([]byte("hello\n"))
	assert.Nil(t, err)

	err = dataFile.Close()
	assert.Nil(t, err)

}

func TestDataFile_Sync(t *testing.T) {
	dataFile, err := OpenDataFile(os.TempDir(), 222, fio.StanderFIO)

	assert.Nil(t, err)
	assert.NotNil(t, dataFile)

	err = dataFile.Write([]byte("hello\n"))
	assert.Nil(t, err)

	err = dataFile.Sync()
	assert.Nil(t, err)
}

func TestDataFile_ReadLogRecord(t *testing.T) {
	dataFile, err := OpenDataFile(os.TempDir(), 999, fio.StanderFIO)
	// dataFile, err := OpenDataFile("./", 888, fio.StanderFIO)

	assert.Nil(t, err)
	assert.NotNil(t, dataFile)

	// 只有一条 LogRecord
	rec1 := &LogRecord{
		Key:   []byte("name"),
		Value: []byte("wsx"),
	}

	// 检测写入和读取出的数据是否一致
	res1, size1 := EncodeLogRecord(rec1)
	err = dataFile.Write(res1)
	assert.Nil(t, err)
	readRec1, readSize1, err := dataFile.ReadLogRecord(0)
	t.Log(size1)
	assert.Nil(t, err)
	assert.Equal(t, rec1, readRec1)
	assert.Equal(t, size1, readSize1)

	// 多条 LogRecord 从不同位置读取
	rec2 := &LogRecord{
		Key:   []byte("age"),
		Value: []byte("23"),
	}
	res2, size2 := EncodeLogRecord(rec2)
	err = dataFile.Write(res2)
	assert.Nil(t, err)
	t.Log(size2)
	readRec2, readSize2, err := dataFile.ReadLogRecord(size1)
	assert.Nil(t, err)
	assert.Equal(t, rec2, readRec2)
	assert.Equal(t, size2, readSize2)

	// 被删除的数据在数据文件的末尾
	rec3 := &LogRecord{
		Key:   []byte("name"),
		Type:  LogRecordDeleted,
		Value: []byte(""),
	}
	res3, size3 := EncodeLogRecord(rec3)
	dataFile.Write(res3)
	assert.Nil(t, err)
	t.Log(size3)
	readRec3, readSize3, err := dataFile.ReadLogRecord(size1 + size2)
	assert.Nil(t, err)
	assert.Equal(t, rec3, readRec3)
	assert.Equal(t, size3, readSize3)

}
