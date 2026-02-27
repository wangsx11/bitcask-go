package data

import (
	"encoding/binary"
	"hash/crc32"
)

type LogRecordType = byte

const (
	LogRecordNormal LogRecordType = iota
	LogRecordDeleted
	LogRecordTxnFinished
)

// crc type keySize valueSize
// 4 + 1 + binary.MaxVarintLen32（5）*2
const maxLogRecordHeaderSize = binary.MaxVarintLen32*2 + 5

// LogRecord 写入到数据文件的记录
// 之所以叫日志 是因为数据文件的数据是追加写入的，类似日志的形式
type LogRecord struct {
	Key   []byte
	Value []byte
	Type  LogRecordType
}

// LogRecordPos 数据内存索引，主要描述数据在磁盘上的位置
type LogRecordPos struct {
	Fid    uint32 // 文件id， 表示将数据存放到了哪个文件中
	Offset int64  // 数据在文件中的偏移量
	Size   uint32  // 数据在磁盘中的大小
}

// transactionRecords 暂存事务相关的数据
type TransactionRecord struct {
	Record *LogRecord
	Pos    *LogRecordPos
}

// LogRecordHeader 数据文件的头信息
type logRecordHeader struct {
	crc        uint32        // crc 校验和
	recordType LogRecordType // 标识LogRecord的类型
	keySize    uint32        // key的大小
	valueSize  uint32        // value的大小
}

// 对LogRecord进行编码，返回字节数组及长度
// +---------------------------------------------------------------------+
// |  crc 校验值  |  type 类型  |  key size  |  value size  | key | value |
// +---------------------------------------------------------------------+
//
//	4字节        1字节       变长(最大5)   变长(最大5)   变长    变长
func EncodeLogRecord(logRecord *LogRecord) ([]byte, int64) {
	// 初始化一个 header 部分的字节数组
	header := make([]byte, maxLogRecordHeaderSize)

	// 第五个字节存储Type
	header[4] = logRecord.Type
	var index = 5
	// 5 字节之后存储的是 key 和 value
	// 使用变长变量，节省空间
	index += binary.PutVarint(header[index:], int64(len(logRecord.Key)))
	index += binary.PutVarint(header[index:], int64(len(logRecord.Value)))

	var size = index + len(logRecord.Key) + len(logRecord.Value)
	encBytes := make([]byte, size)

	// 将 header 部分拷贝过来
	copy(encBytes[:index], header[:index])
	// 将 key 和 value 拷贝过来
	copy(encBytes[index:], logRecord.Key)
	copy(encBytes[index+len(logRecord.Key):], logRecord.Value)

	// 对整个 logRecord 进行CRC校验
	crc := crc32.ChecksumIEEE(encBytes[4:])
	binary.LittleEndian.PutUint32(encBytes[:4], crc)

	return encBytes, int64(size)
}

// 对位置信息进行编码
func EncodeLogRecordPos(pos *LogRecordPos) []byte {
	// 创建一个长度为 12 的字节数组
	buf := make([]byte, binary.MaxVarintLen32*2+binary.MaxVarintLen64)
	var index = 0
	index += binary.PutVarint(buf[index:], int64(pos.Fid))
	index += binary.PutVarint(buf[index:], pos.Offset)
	index += binary.PutVarint(buf[index:], int64(pos.Size))

	return buf[:index]
}

// 解码 LogRecordPos
func DecodeLogRecordPos(buf []byte) *LogRecordPos {
	var index = 0
	fid, n := binary.Varint(buf[index:])
	index += n
	offset, n := binary.Varint(buf[index:])
	index += n
	size, _ := binary.Varint(buf[index:])
	return &LogRecordPos{
		Fid:    uint32(fid),
		Offset: offset,
		Size:   uint32(size),
	}
}

// 对字节数组中的 Header 信息进行解码
func DecodeLogRecordHeader(buf []byte) (*logRecordHeader, int64) {
	// 小于等于 CRC 的字节数直接返回
	if len(buf) <= 4 {
		return nil, int64(0)
	}
	header := &logRecordHeader{
		crc:        binary.LittleEndian.Uint32(buf[:4]),
		recordType: buf[4],
	}
	var index = 5
	// 取出实际的 key/value size
	keySize, n := binary.Varint(buf[index:])
	header.keySize = uint32(keySize)
	index += n
	ValueSize, n := binary.Varint(buf[index:])
	header.valueSize = uint32(ValueSize)
	index += n

	return header, int64(index)
}

func GetLogRecordCRC(lr *LogRecord, header []byte) uint32 {
	if lr == nil {
		return 0
	}
	crc := crc32.ChecksumIEEE(header[:])
	crc = crc32.Update(crc, crc32.IEEETable, lr.Key)
	crc = crc32.Update(crc, crc32.IEEETable, lr.Value)

	return crc
}
