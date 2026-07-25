package index

import (
	"bitcask-go/data"
	"bytes"
	"sort"
	"sync"

	goart "github.com/plar/go-adaptive-radix-tree"
)

// 自适应基数树
// 主要封装了 https://github.com/plar/go-adaptive-radix-tree

type AdaptiveRadixTree struct {
	tree goart.Tree
	lock *sync.RWMutex
}

func NewART() *AdaptiveRadixTree {
	return &AdaptiveRadixTree{
		tree: goart.New(),
		lock: new(sync.RWMutex),
	}
}

func (art *AdaptiveRadixTree) Put(key []byte, pos *data.LogRecordPos) (*data.LogRecordPos, error) {
	art.lock.Lock()
	oldValue, _ := art.tree.Insert(key, pos)
	art.lock.Unlock()
	if oldValue == nil {
		return nil, nil
	}
	// fmt.Println("oldValue:", oldValue)
	return oldValue.(*data.LogRecordPos), nil
}

func (art *AdaptiveRadixTree) Get(key []byte) (*data.LogRecordPos, error) {
	art.lock.RLock()
	defer art.lock.RUnlock()

	value, ok := art.tree.Search(key)
	if !ok {
		return nil, nil
	}
	return value.(*data.LogRecordPos), nil // 这里返回的是interface{}, 需要转换成 LogRecordPos
}

func (art *AdaptiveRadixTree) Delete(key []byte) (*data.LogRecordPos, bool, error) {
	art.lock.Lock()
	oldValue, deleted := art.tree.Delete(key)
	art.lock.Unlock()
	if oldValue == nil {
		return nil, false, nil
	}
	return oldValue.(*data.LogRecordPos), deleted, nil
}

func (art *AdaptiveRadixTree) Size() (int, error) {
	art.lock.RLock()
	size := art.tree.Size()
	art.lock.RUnlock()
	return size, nil
}

func (art *AdaptiveRadixTree) Iterator(reverse bool) (Iterator, error) {
	art.lock.RLock()
	defer art.lock.RUnlock()
	return newARTIterator(art.tree, reverse), nil
}

func (art *AdaptiveRadixTree) Close() error {
	return nil
}

// Art 索引迭代器
type artIterator struct {
	currIndex int     // 当前遍历的下标位置
	reverse   bool    // 是否为反向遍历
	values    []*Item // key + 位置索引信息
}

func newARTIterator(tree goart.Tree, reverse bool) *artIterator {
	var idx int
	if reverse {
		idx = tree.Size() - 1
	}
	values := make([]*Item, tree.Size())
	saveValues := func(node goart.Node) bool {

		item := &Item{
			key: node.Key(),
			pos: node.Value().(*data.LogRecordPos),
		}
		values[idx] = item
		if reverse {
			idx--
		} else {
			idx++
		}
		return true
	}

	// tree.ForEach 会遍历树中的所有节点，通过reverse决定存储位置
	tree.ForEach(saveValues)

	return &artIterator{
		currIndex: 0,
		reverse:   reverse,
		values:    values,
	}
}

func (ai *artIterator) Rewind() {
	ai.currIndex = 0
}

func (ai *artIterator) Seek(key []byte) {
	if ai.reverse {
		ai.currIndex = sort.Search(len(ai.values), func(i int) bool {
			return bytes.Compare(ai.values[i].key, key) <= 0
		})
	} else {
		ai.currIndex = sort.Search(len(ai.values), func(i int) bool {
			return bytes.Compare(ai.values[i].key, key) >= 0
		})
	}
}
func (ai *artIterator) Next() {
	ai.currIndex += 1

}
func (ai *artIterator) Valid() bool {
	return ai.currIndex < len(ai.values)
}
func (ai *artIterator) Key() []byte {
	return ai.values[ai.currIndex].key
}
func (ai *artIterator) Value() *data.LogRecordPos {
	return ai.values[ai.currIndex].pos

}
func (ai *artIterator) Close() error {
	ai.values = nil
	return nil
}
