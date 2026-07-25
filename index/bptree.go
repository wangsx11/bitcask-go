package index

import (
	"bitcask-go/data"
	"errors"
	"path/filepath"

	"go.etcd.io/bbolt"
)

const bptreeIndexFileName = "bptree-index"

var indexBucketName = []byte("bitcask-index")

// B+ 树索引
// 主要封装了 go.etcd.io/bbolt 库的接口

type BPlusTree struct {
	// bbolt 支持并发访问
	tree *bbolt.DB
}

func NewBPlusTree(dirPath string, syncWrites bool) (*BPlusTree, error) {
	opts := *bbolt.DefaultOptions // 复制默认配置，避免修改 bbolt 的共享配置
	opts.NoSync = !syncWrites
	bptree, err := bbolt.Open(filepath.Join(dirPath, bptreeIndexFileName), 0644, &opts)
	if err != nil {
		return nil, err
	}

	// 创建对应的 bucket
	if err := bptree.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(indexBucketName)
		return err
	}); err != nil {
		return nil, errors.Join(err, bptree.Close())
	}

	return &BPlusTree{
		tree: bptree,
	}, nil
}

func (bpt *BPlusTree) Put(key []byte, pos *data.LogRecordPos) (*data.LogRecordPos, error) {
	var oldVal []byte
	if err := bpt.tree.Update(func(tx *bbolt.Tx) error {

		bucket := tx.Bucket(indexBucketName)
		oldVal = bucket.Get(key)
		return bucket.Put(key, data.EncodeLogRecordPos(pos))
	}); err != nil {
		return nil, err
	}
	if len(oldVal) == 0 {
		return nil, nil
	}
	return data.DecodeLogRecordPos(oldVal), nil
}

func (bpt *BPlusTree) Get(key []byte) (*data.LogRecordPos, error) {
	var pos *data.LogRecordPos
	if err := bpt.tree.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(indexBucketName)
		value := bucket.Get(key)
		if len(value) != 0 {
			pos = data.DecodeLogRecordPos(value)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return pos, nil
}

func (bpt *BPlusTree) Delete(key []byte) (*data.LogRecordPos, bool, error) {
	var oldVal []byte
	if err := bpt.tree.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(indexBucketName)
		if oldVal = bucket.Get(key); len(oldVal) != 0 {
			return bucket.Delete(key)
		}
		return nil
	}); err != nil {
		return nil, false, err
	}
	if len(oldVal) == 0 {
		return nil, false, nil
	}
	return data.DecodeLogRecordPos(oldVal), true, nil
}

func (bpt *BPlusTree) Size() (int, error) {
	var size int
	if err := bpt.tree.View(func(tx *bbolt.Tx) error {
		buctet := tx.Bucket(indexBucketName)
		size = buctet.Stats().KeyN
		return nil
	}); err != nil {
		return 0, err
	}
	return size, nil
}

func (bpt *BPlusTree) Iterator(reverse bool) (Iterator, error) {
	return newBptreeIterator(bpt.tree, reverse)
}

func (bpt *BPlusTree) Close() error {
	return bpt.tree.Close()
}

// B+ 树迭代器
type bptreeIterator struct {
	tx        *bbolt.Tx
	cursor    *bbolt.Cursor
	reverse   bool
	currKey   []byte
	currValue []byte
}

func newBptreeIterator(tree *bbolt.DB, reverse bool) (*bptreeIterator, error) {
	tx, err := tree.Begin(false)
	if err != nil {
		return nil, err
	}
	bpi := &bptreeIterator{
		tx:      tx,
		cursor:  tx.Bucket(indexBucketName).Cursor(),
		reverse: reverse,
	}
	// 需要显示调用Rewind，否则currKey以及currValue为空，调用Vaild会报错
	bpi.Rewind()
	return bpi, nil
}

func (bpi *bptreeIterator) Rewind() {
	if bpi.reverse {
		bpi.currKey, bpi.currValue = bpi.cursor.Last()
	} else {
		bpi.currKey, bpi.currValue = bpi.cursor.First()
	}
}

func (bpi *bptreeIterator) Seek(key []byte) {
	bpi.currKey, bpi.currValue = bpi.cursor.Seek(key)
}

func (bpi *bptreeIterator) Next() {
	if bpi.reverse {
		bpi.currKey, bpi.currValue = bpi.cursor.Prev()
	} else {
		bpi.currKey, bpi.currValue = bpi.cursor.Next()
	}
}

func (bpi *bptreeIterator) Valid() bool {
	return len(bpi.currKey) != 0
}

func (bpi *bptreeIterator) Key() []byte {
	return bpi.currKey

}

func (bpi *bptreeIterator) Value() *data.LogRecordPos {
	return data.DecodeLogRecordPos(bpi.currValue)
}

func (bpi *bptreeIterator) Close() error {
	return bpi.tx.Rollback() // 只读使用rollback而不是commit
}
