package index

import (
	"bitcask-go/data"
	"errors"
)

var ErrUnsupportedIndexType = errors.New("unsupported index type")

// Indexer 抽象索引接口， 后续如果想要接入其他的数据结构，则直接实现这个接口即可
type Indexer interface {
	// Put 向索引中存储 key 对应的数据位置信息
	Put(key []byte, pos *data.LogRecordPos) (*data.LogRecordPos, error)

	// Get 根据 key 获取数据位置信息
	Get(key []byte) (*data.LogRecordPos, error)

	// Delete 根据 key 删除数据
	Delete(key []byte) (*data.LogRecordPos, bool, error)

	// Size 索引中的数据量
	Size() (int, error)

	// Iterator 索引迭代器
	Iterator(reverse bool) (Iterator, error)

	// Close 关闭索引
	Close() error
}
type IndexType = int8

const (
	// BTree 索引
	Btree IndexType = iota + 1

	// ART 索引 自适应基数树
	ART

	// B+Tree 索引
	BPTree
)

func NewIndexer(typ IndexType, dirPath string, sync bool) (Indexer, error) {
	switch typ {
	case Btree:
		return NewBTree(), nil
	case ART:
		return NewART(), nil
	case BPTree:
		return NewBPlusTree(dirPath, sync)
	default:
		return nil, ErrUnsupportedIndexType
	}
}

// Item 索引项，存储键和对应的数据位置信息
// 为所有索引实现提供统一的数据结构
type Item struct {
	key []byte
	pos *data.LogRecordPos
}

// Indexer 抽象索引接口，后续如果想要接入其他的数据结构，则直接实现这个接口即可
type Iterator interface {
	// 重新回到迭代器的起点，及第一个元素
	Rewind()
	// 根据传入的 key 查找到第一个大于或小于等于的目标 key，从这个 key 开始遍历
	Seek(key []byte)
	// 跳转到下一个 key
	Next()
	// 是否有效，即是否已经遍历完了所有的 key，用于退出遍历
	Valid() bool
	// 当前遍历位置的 key 数据
	Key() []byte
	// 当前遍历位置的 value 数据
	Value() *data.LogRecordPos
	// 关闭迭代器，释放相应资源
	Close() error
}
