package data

import (
	"hash/crc32"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeLogRecord(t *testing.T) {
	// 正常情况
	rec1 := &LogRecord{
		Key:   []byte("name"),
		Type:  LogRecordNormal,
		Value: []byte("wsx"),
	}
	res1, n1 := EncodeLogRecord(rec1)
	t.Log(res1)

	assert.NotNil(t, res1)
	assert.Greater(t, n1, int64(5))

	// value 为空
	rec2 := &LogRecord{
		Key:  []byte("name"),
		Type: LogRecordNormal,
	}
	res2, n2 := EncodeLogRecord(rec2)
	assert.NotNil(t, res2)
	assert.Greater(t, n2, int64(5))

	// 对 Deleteed 情况的测试
	rec3 := &LogRecord{
		Key:  []byte("name"),
		Type: LogRecordDeleted,
	}
	res3, n3 := EncodeLogRecord(rec3)
	assert.NotNil(t, res3)
	assert.Greater(t, n3, int64(5))
	t.Log(res3)
	t.Log(n3)

}

func TestDecodeLogRecord(t *testing.T) {
	// 对应 TestEncodeLogRecord 正常情况
	headerBuf1 := []byte{141, 49, 52, 224, 0, 8, 6}
	h1, size1 := DecodeLogRecordHeader(headerBuf1)
	assert.NotNil(t, h1)
	assert.Equal(t, int64(7), size1)
	assert.Equal(t, uint32(3761516941), h1.crc)
	assert.Equal(t, LogRecordNormal, h1.recordType)
	assert.Equal(t, uint32(4), h1.keySize)
	assert.Equal(t, uint32(3), h1.valueSize)

	// value 为空
	headerBuf2 := []byte{9, 252, 88, 14, 0, 8, 0}
	h2, size2 := DecodeLogRecordHeader(headerBuf2)
	assert.NotNil(t, h2)
	assert.Equal(t, int64(7), size2)
	assert.Equal(t, uint32(240712713), h2.crc)
	assert.Equal(t, LogRecordNormal, h2.recordType)
	assert.Equal(t, uint32(4), h2.keySize)
	assert.Equal(t, uint32(0), h2.valueSize)

	// 对 Deleteed 情况的测试
	headerBuf3 := []byte{189, 247, 47, 168, 1, 8, 0}
	h3, size3 := DecodeLogRecordHeader(headerBuf3)
	assert.NotNil(t, h3)
	assert.Equal(t, int64(7), size3)
	assert.Equal(t, uint32(2821715901), h3.crc)
	assert.Equal(t, LogRecordDeleted, h3.recordType)
	assert.Equal(t, uint32(4), h3.keySize)
	assert.Equal(t, uint32(0), h3.valueSize)

}

func TestGetLogRecordCRC(t *testing.T) {
	rec1 := &LogRecord{
		Key:   []byte("name"),
		Type:  LogRecordNormal,
		Value: []byte("wsx"),
	}
	headerBuf1 := []byte{141, 49, 52, 224, 0, 8, 6}
	crc1 := GetLogRecordCRC(rec1, headerBuf1[crc32.Size:])
	assert.Equal(t, uint32(3761516941), crc1)

	// value 为空
	rec2 := &LogRecord{
		Key:  []byte("name"),
		Type: LogRecordNormal,
	}
	headerBuf2 := []byte{9, 252, 88, 14, 0, 8, 0}
	crc2 := GetLogRecordCRC(rec2, headerBuf2[crc32.Size:])
	t.Log(crc2)
	assert.Equal(t, uint32(240712713), crc2)

	// 对 Deleteed 情况的测试
	rec3 := &LogRecord{
		Key:  []byte("name"),
		Type: LogRecordDeleted,
	}
	headerBuf3 := []byte{189, 247, 47, 168, 1, 8, 0}
	crc3 := GetLogRecordCRC(rec3, headerBuf3[crc32.Size:])
	t.Log(crc3)
	assert.Equal(t, uint32(2821715901), crc3)

}
