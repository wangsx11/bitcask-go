package index

import (
	"bitcask-go/data"
	"bytes"
	"sort"

	"sync"

	"github.com/google/btree"
)

// BTree 索引，主要封装了google的btree kv
type BTree struct {
	tree *btree.BTree // 并发写不安全 并发读安全
	lock *sync.RWMutex
}

// 初始化 BTree 索引结构
func NewBTree() *BTree {
	return &BTree{
		tree: btree.New(32), // BTree 叶子节点的数量 后续可以改成一个参数让用户自行选择
		lock: new(sync.RWMutex),
	}
}

// Less 实现btree.Item接口，为BTree提供节点比较功能
func (ai *Item) Less(bi btree.Item) bool { // key 的比较规则
	return bytes.Compare(ai.key, bi.(*Item).key) == -1
}

func (bt *BTree) Put(key []byte, pos *data.LogRecordPos) (*data.LogRecordPos, error) {
	it := &Item{key: key, pos: pos}
	bt.lock.Lock()
	oldItem := bt.tree.ReplaceOrInsert(it)
	bt.lock.Unlock()
	if oldItem == nil {
		return nil, nil
	}
	return oldItem.(*Item).pos, nil
}

func (bt *BTree) Get(key []byte) (*data.LogRecordPos, error) {
	it := &Item{key: key}
	btreeItem := bt.tree.Get(it) // 这里返回的btreeItem是一个interface{} 所以后续还需要进行断言
	if btreeItem == nil {
		return nil, nil
	}
	return btreeItem.(*Item).pos, nil
}
func (bt *BTree) Delete(key []byte) (*data.LogRecordPos, bool, error) {
	it := &Item{key: key}
	bt.lock.Lock()
	oldItem := bt.tree.Delete(it)
	bt.lock.Unlock()
	if oldItem == nil {
		return nil, false, nil
	}
	return oldItem.(*Item).pos, true, nil
}

func (bt *BTree) Size() (int, error) {
	if bt.tree == nil {
		return 0, nil
	}
	bt.lock.RLock()
	defer bt.lock.RUnlock()
	return bt.tree.Len(), nil
}
func (bt *BTree) Iterator(reverse bool) (Iterator, error) {
	if bt.tree == nil {
		return nil, nil
	}
	bt.lock.RLock()
	defer bt.lock.RUnlock()
	return newBtreeIterator(bt.tree, reverse), nil
}

func (bt *BTree) Close() error {
	return nil
}

// BTree 索引迭代器
type btreeIterator struct {
	currIndex int     // 当前遍历的下标位置
	reverse   bool    // 是否为反向遍历
	values    []*Item // key + 位置索引信息
}

func newBtreeIterator(tree *btree.BTree, reverse bool) *btreeIterator {
	var idx int
	values := make([]*Item, tree.Len())

	// 将所有的数据保存到数组中
	saveValues := func(it btree.Item) bool {
		values[idx] = it.(*Item)
		idx++
		return true
	}
	if reverse {
		tree.Descend(saveValues)
	} else {
		tree.Ascend(saveValues)
	}

	return &btreeIterator{
		currIndex: 0,
		reverse:   reverse,
		values:    values,
	}
}

func (bti *btreeIterator) Rewind() {
	bti.currIndex = 0
}

// Seek 将迭代器定位到指定key的位置，确保后续遍历能获取到所有匹配前缀的key
// 正向遍历：定位到第一个 >= key 的位置，向后遍历更大的key
// 反向遍历：定位到第一个 <= key 的位置，向前遍历更小的key
func (bti *btreeIterator) Seek(key []byte) {
	if bti.reverse {
		bti.currIndex = sort.Search(len(bti.values), func(i int) bool {
			return bytes.Compare(bti.values[i].key, key) <= 0
		})
	} else {
		bti.currIndex = sort.Search(len(bti.values), func(i int) bool {
			return bytes.Compare(bti.values[i].key, key) >= 0
		})
	}
}
func (bti *btreeIterator) Next() {
	// TODO 如果是反向 这个地方不是应该减一吗
	// --> 因为在 newBtreeIterator 时， 如果是反向遍历，已经将数据倒序存储了， 所以都是+1
	bti.currIndex += 1
}
func (bti *btreeIterator) Valid() bool {
	return bti.currIndex < len(bti.values)
}
func (bti *btreeIterator) Key() []byte {
	return bti.values[bti.currIndex].key
}
func (bti *btreeIterator) Value() *data.LogRecordPos {
	return bti.values[bti.currIndex].pos
}
func (bti *btreeIterator) Close() error {
	bti.values = nil
	return nil
}
