package bitcask_go

import (
	"bitcask-go/index"
	"bytes"
)

// Iterator 迭代器
type Iterator struct {
	indexIter index.Iterator // 索引迭代器
	db        *DB
	options   IteratorOptions
}

func (db *DB) NewIterator(opts IteratorOptions) (*Iterator, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	if db.closed {
		return nil, ErrClosed
	}
	indexIter, err := db.index.Iterator(opts.Reverse)
	if err != nil {
		return nil, err
	}
	return &Iterator{
		db:        db,
		indexIter: indexIter,
		options:   opts,
	}, nil
}

func (it *Iterator) Rewind() {
	it.indexIter.Rewind()
	it.skipToNext()
}
func (it *Iterator) Seek(key []byte) {
	it.indexIter.Seek(key)
	it.skipToNext()
}

func (it *Iterator) Next() {
	it.indexIter.Next()
	it.skipToNext()
}
func (it *Iterator) Valid() bool {
	return it.indexIter.Valid()
}
func (it *Iterator) Key() []byte {
	return it.indexIter.Key()
}
func (it *Iterator) Value() ([]byte, error) {
	logrecordPos := it.indexIter.Value()
	it.db.mu.RLock()
	defer it.db.mu.RUnlock()
	if it.db.closed {
		return nil, ErrClosed
	}
	return it.db.getValueByPosition(logrecordPos)
}
func (it *Iterator) Close() error {
	return it.indexIter.Close()
}
func (it *Iterator) skipToNext() {
	prefixLen := len(it.options.Prefix)
	if prefixLen == 0 {
		return
	}
	for ; it.indexIter.Valid(); it.indexIter.Next() {
		key := it.indexIter.Key()
		// if prefixLen <= len(key) && bytes.Compare(it.options.Prefix, key[:prefixLen]) == 0 {
		// 	break
		// }
		if prefixLen <= len(key) && bytes.Equal(it.options.Prefix, key[:prefixLen]) {
			break
		}
	}
}
