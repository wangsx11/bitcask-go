package data

import (
	"bitcask-go/fio"
	"testing"

	"github.com/stretchr/testify/assert"
)

func openTestDataFile(t *testing.T, fileID uint32) *DataFile {
	t.Helper()
	dataFile, err := OpenDataFile(t.TempDir(), fileID, fio.StanderFIO)
	assert.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, dataFile.Close()) })
	return dataFile
}

func TestOpenDataFile(t *testing.T) {
	dataFile := openTestDataFile(t, 0)
	assert.NotNil(t, dataFile)
}

func TestDataFileWrite(t *testing.T) {
	dataFile := openTestDataFile(t, 0)

	for _, value := range [][]byte{
		[]byte("hello\n"),
		[]byte("go\n"),
		[]byte("nice to meet you\n"),
		[]byte("nice to meet you too\n"),
	} {
		assert.NoError(t, dataFile.Write(value))
	}
}

func TestDataFileClose(t *testing.T) {
	dataFile, err := OpenDataFile(t.TempDir(), 0, fio.StanderFIO)
	assert.NoError(t, err)
	assert.NoError(t, dataFile.Write([]byte("hello\n")))
	assert.NoError(t, dataFile.Close())
}

func TestDataFileSync(t *testing.T) {
	dataFile := openTestDataFile(t, 222)
	assert.NoError(t, dataFile.Write([]byte("hello\n")))
	assert.NoError(t, dataFile.Sync())
}

func TestDataFileReadLogRecord(t *testing.T) {
	dataFile := openTestDataFile(t, 999)
	records := []*LogRecord{
		{Key: []byte("name"), Value: []byte("wsx")},
		{Key: []byte("age"), Value: []byte("23")},
		{Key: []byte("name"), Value: []byte{}, Type: LogRecordDeleted},
	}

	var offset int64
	for _, record := range records {
		encoded, size := EncodeLogRecord(record)
		assert.NoError(t, dataFile.Write(encoded))

		actual, actualSize, err := dataFile.ReadLogRecord(offset)
		assert.NoError(t, err)
		assert.Equal(t, record, actual)
		assert.Equal(t, size, actualSize)
		offset += size
	}
}
