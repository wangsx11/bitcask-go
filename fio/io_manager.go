package fio

const DataFilePerm = 0644

type FileIOType = byte

const (
	// 标准文件 IO
	StanderFIO = iota
	// 内存文件映射
	MemoryMap
)

// 抽象 IO 管理接口，可以接入不同的 IO 类型， 目前支持标准文件 IO
type IOManager interface {
	// Read 从文件的指定位置读取对应数据
	Read([]byte, int64) (int, error)

	// Write 写入字节数据到文件中 只支持追加写入
	Write([]byte) (int, error)

	// Sync 强制将文件数据刷入磁盘
	Sync() error

	// Close 关闭文件
	Close() error

	// 获取文件大小
	Size() (int64, error)
}
	
// 初始化 IOManager 目前只支持FileIO
func NewIOManager(fileName string, ioType FileIOType) (IOManager, error) {
	switch ioType {
	case StanderFIO:
		return NewFileIOManager(fileName)
	case MemoryMap:
		return NewMMapIOManager(fileName)
	default:
		panic("unsupported io type")
	}
}
