package fio

import (
	"os"
)

// FileIO 标准系统文件 IO
type FileIO struct {
	fd *os.File // 系统文件描述符
}

// NewFileManager 初始化标准文件 IO
func NewFileIOManager(fileName string) (*FileIO, error) {
	// 如果文件不存在就创建、赋予读写权限、且只允许追加写入
	fd, err := os.OpenFile(
		fileName,
		os.O_CREATE|os.O_RDWR|os.O_APPEND,
		DataFilePerm,
	)
	if err != nil {
		return nil, err
	}
	return &FileIO{fd: fd}, nil
}

func (fio *FileIO) Read(b []byte, offset int64) (int, error) {
	return fio.fd.ReadAt(b, offset)
}

// Write 写入字节数据到文件中 只支持追加写入
func (fio *FileIO) Write(b []byte) (int, error) {
	return fio.fd.Write(b)
}

// Sync 强制将文件数据刷入磁盘
func (fio *FileIO) Sync() error {
	return fio.fd.Sync()

}

// Close 关闭文件
func (fio *FileIO) Close() error {
	return fio.fd.Close()

}

// Size 获取文件大小
func (fio *FileIO) Size() (int64, error) {
	stat, err := fio.fd.Stat()
	if err != nil {
		return 0, err
	}
	return stat.Size(), nil
}
