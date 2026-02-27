package data

import (
	"bitcask-go/fio"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"path/filepath"
)

var (
	ErrinvalidCRC = errors.New("invalid crc value, log record mabey corrupted")
)

const (
	DataFileNameSuffix    = ".data"
	HintFileName          = "hint-index"
	MergeFinishedFileName = "merge-finished"
	SeqNoFileName         = "seq-no"
)

// DataFile 数据文件
type DataFile struct {
	FileId    uint32        //文件id
	WriteOff  int64         // 文件写到了那个位置
	IoManager fio.IOManager // 读写管理
}

// 打开新的数据文件
// 这里的fileId 怎么大都是0
func OpenDataFile(dirPath string, fileId uint32, ioType fio.FileIOType) (*DataFile, error) {
	fileName := GetDataFileName(dirPath, fileId)
	return newDatafile(fileName, fileId, ioType)

}

// 打开 Hint 索引文件
func OpenHintFile(dirPath string) (*DataFile, error) {
	fileName := filepath.Join(dirPath, HintFileName)
	return newDatafile(fileName, 0,fio.StanderFIO)
}

// 标识 merge 完成
func OpenMergeFinishedFile(dirPath string) (*DataFile, error) {
	fileName := filepath.Join(dirPath, MergeFinishedFileName)
	return newDatafile(fileName, 0, fio.StanderFIO)
}

// 存储seq序列号
func OpenSeqNoFile(dirPath string) (*DataFile, error) {
	fileName := filepath.Join(dirPath, SeqNoFileName)
	return newDatafile(fileName, 0, fio.StanderFIO)
}

func GetDataFileName(dirPath string, fileId uint32) string {
	return filepath.Join(dirPath, fmt.Sprintf("%09d", fileId)+DataFileNameSuffix)
}

func newDatafile(fileName string, fileId uint32, ioType fio.FileIOType) (*DataFile, error) {
	// 初始化 IOManager 管理器接口
	ioManager, err := fio.NewIOManager(fileName, ioType)
	if err != nil {
		return nil, err
	}

	return &DataFile{
		FileId:    fileId,
		WriteOff:  0,
		IoManager: ioManager,
	}, nil
}

func (df *DataFile) Sync() error {
	return df.IoManager.Sync()
}

func (df *DataFile) Close() error {
	return df.IoManager.Close()
}

func (df *DataFile) Write(buf []byte) error {
	n, err := df.IoManager.Write(buf)
	if err != nil {
		return err
	}
	df.WriteOff += int64(n)
	return nil
}

// 写入索引信息到 hint 文件中
func (df *DataFile) WriteHintRecord(key []byte, pos *LogRecordPos) error {
	record := &LogRecord{
		Key:   key,
		Value: EncodeLogRecordPos(pos),
	}
	encRecord, _ := EncodeLogRecord(record)
	return df.Write(encRecord)
}

// 根据 offset 从数据文件中读取 LogRecord
func (df *DataFile) ReadLogRecord(offset int64) (*LogRecord, int64, error) {
	fileSize, err := df.IoManager.Size()
	if err != nil {
		return nil, 0, err
	}

	var headerBytes int64 = maxLogRecordHeaderSize
	// 如果读取到的最大 header 长度已经超过了文件的长度，则只需要读取到文件的末尾即可
	if offset+headerBytes > fileSize {
		headerBytes = fileSize - offset
	}
	// 从文件的offset处读取headerBytes个字节
	headerBuf, err := df.ReadNBytes(headerBytes, offset)
	if err != nil {
		return nil, 0, err
	}

	header, headerSize := DecodeLogRecordHeader(headerBuf)

	// 下面两个条件表示读取到了文件末尾，直接返回 EOF错误
	if header == nil {
		return nil, 0, io.EOF
	}
	if header.crc == 0 && header.keySize == 0 && header.valueSize == 0 {
		return nil, 0, io.EOF
	}

	// 取出 key 和 value 的长度
	keySize, valueSize := int64(header.keySize), int64(header.valueSize)
	var recordSize = headerSize + keySize + valueSize

	// 开始读取用户实际存储的 key/value 数据
	logRecord := &LogRecord{
		Type: header.recordType,
	}
	if keySize > 0 || valueSize > 0 {
		kvBuf, err := df.ReadNBytes(keySize+valueSize, offset+headerSize)
		if err != nil {
			return nil, 0, err
		}
		// 解出 key 和 value
		logRecord.Key = kvBuf[:keySize]
		logRecord.Value = kvBuf[keySize:]
	}

	// 校验数据的有效性
	crc := GetLogRecordCRC(logRecord, headerBuf[crc32.Size:headerSize])
	if crc != header.crc {
		return nil, 0, ErrinvalidCRC
	}

	return logRecord, recordSize, nil
}

func (df *DataFile) ReadNBytes(n int64, offset int64) (b []byte, err error) {
	b = make([]byte, n)
	_, err = df.IoManager.Read(b, offset)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (df *DataFile) SetIOManager(dirpath string, ioType fio.FileIOType) error {
	if err := df.IoManager.Close(); err != nil {
		return err
	}
	ioManager, err := fio.NewIOManager(GetDataFileName(dirpath, df.FileId), ioType)
	if err != nil {
		return err
	}
	df.IoManager = ioManager
	return nil
}
