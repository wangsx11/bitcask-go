package bitcask_go

import "os"

type Options struct {
	DirPath            string      // 数据库数据目录
	DataFileSize       int64       // 活跃文件大小
	SyncWrites         bool        // 每次写入是否需要持久化
	BytesPerSync       uint        // 累计写入到多少字节后进行持久化
	IndexerType        IndexerType // 索引类型
	MMapAtStartup      bool        // 启动时是否需要使用 MMap 加载数据
	DataFileMergeRatio float32     // 数据文件合并的阈值
}

// IteratorOptions 索引迭代器配置项
type IteratorOptions struct {
	// 遍历前缀为指定值的 key， 默认为空
	Prefix []byte

	// 是否反向遍历，默认为 false 是正向
	Reverse bool
}

// WriteBatchOptions 批量写入配置项
type WriteBatchOptions struct {
	MaxBatchNum uint // 一个批次中最大的数据量
	SyncWrites  bool // 提交时是否将数据持久化
}

type IndexerType = int8

const (
	// BTree 索引
	BTree IndexerType = iota + 1

	// ART 自适应基数树索引
	ART

	// B+ 树索引 将索引存储到磁盘中
	BPlusTree
)

// DefaultOptions returns an independent default configuration.
func DefaultOptions() Options {
	return Options{
		DirPath:            os.TempDir(),
		DataFileSize:       256 * 1024 * 1024, // 256MB
		SyncWrites:         false,
		BytesPerSync:       0,
		IndexerType:        ART,
		MMapAtStartup:      false,
		DataFileMergeRatio: 0.5,
	}
}

var DefaultIteratorOptions = IteratorOptions{
	Prefix:  nil,
	Reverse: false,
}

var DefaultWriteBatchOptions = WriteBatchOptions{
	MaxBatchNum: 10000,
	SyncWrites:  true,
}
