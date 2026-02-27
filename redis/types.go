package redis

import (
	bitcask "bitcask-go"
	"bitcask-go/utils"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"time"
)

var (
	ErrWrongTypeOperation = errors.New("WRONG TYPE OPERATION against a key holding the wrong kind of value")
	ErrKeyExpired         = errors.New("EXPIRED: key has reached its time-to-live and is no longer valid")
)

type redisDataType = byte

const (
	String redisDataType = iota
	Hash
	Set
	List
	ZSet
)

// Redis 数据结构服务
type RedisDataStructure struct {
	db *bitcask.DB
}

// 初始化Redis 数据结构服务
func NewRedisDataStructure(options *bitcask.Options) (*RedisDataStructure, error) {
	db, err := bitcask.Open(options)
	if err != nil {
		return nil, err
	}
	return &RedisDataStructure{
		db: db,
	}, nil
}

// ++++++++++++++++++++++++++++ string 数据结构 ++++++++++++++++++++++++++++
func (rds *RedisDataStructure) Set(key []byte, ttl time.Duration, value []byte) error {
	if value == nil {
		return nil
	}
	// 编码 value : type + expire + payload
	buf := make([]byte, binary.MaxVarintLen64+1)
	buf[0] = String
	var index = 1
	var expire int64 = 0
	if ttl != 0 {
		expire = time.Now().Add(ttl).UnixNano()
	}
	index += binary.PutVarint(buf[index:], expire)
	encValue := make([]byte, index+len(value))
	copy(encValue[:index], buf[:index])
	copy(encValue[index:], value)

	// 调用存储接口写入数据
	return rds.db.Put(key, encValue)
}

func (rds *RedisDataStructure) Get(key []byte) ([]byte, error) {
	encValue, err := rds.db.Get(key)
	if err != nil {
		return nil, err
	}
	// 解码 value
	dataType := encValue[0]
	if dataType != String {
		return nil, ErrWrongTypeOperation
	}
	var index = 1
	expire, n := binary.Varint(encValue[index:])
	index += n
	// 判断是否过期
	if expire > 0 && expire < time.Now().UnixNano() {
		return nil, ErrKeyExpired
	}
	return encValue[index:], nil
}

// ++++++++++++++++++++++++++++ Hash 数据结构 ++++++++++++++++++++++++++++

func (rds *RedisDataStructure) HSet(key, field, value []byte) (bool, error) {
	// 获取元数据
	meta, err := rds.findMetadata(key, Hash)
	if err != nil {
		return false, err
	}

	// 构造 hash 数据部分的 Key
	hk := &hashInternalKey{
		key:     key,
		version: meta.version,
		field:   field,
	}
	encKey := hk.enocde()

	// 查找数据是否存在
	var exist = true
	if _, err := rds.db.Get(encKey); err == bitcask.ErrKeyNotFound {
		exist = false
	}
	wb := rds.db.NewWriteBatch(bitcask.DefaultWriteBatchOptions)
	// 不存在则更新元数据
	if !exist {
		meta.size++
		wb.Put(key, meta.encode())
	}
	wb.Put(encKey, value)

	if err = wb.Commit(); err != nil {
		return false, err
	}
	return !exist, nil
}

func (rds *RedisDataStructure) HGet(key, field []byte) ([]byte, error) {
	meta, err := rds.findMetadata(key, Hash)
	if err != nil {
		return nil, err
	}
	if meta.size == 0 {
		return nil, nil
	}

	hk := hashInternalKey{
		key:     key,
		version: meta.version,
		field:   field,
	}
	encKey := hk.enocde()
	return rds.db.Get(encKey)
}

func (rds *RedisDataStructure) HDel(key, field []byte) (bool, error) {
	meta, err := rds.findMetadata(key, Hash)
	if err != nil {
		return false, err
	}
	if meta.size == 0 {
		return false, nil
	}
	hk := hashInternalKey{
		key:     key,
		version: meta.version,
		field:   field,
	}
	encKey := hk.enocde()
	// 查看是否存在
	var exist = true
	if _, err := rds.db.Get(encKey); err == bitcask.ErrKeyNotFound {
		exist = false
	}
	if exist {
		wb := rds.db.NewWriteBatch(bitcask.DefaultWriteBatchOptions)
		meta.size--
		_ = wb.Put(key, meta.encode())
		_ = wb.Delete(encKey)
		if err = wb.Commit(); err != nil {
			return false, err
		}
	}
	return exist, nil
}

func (rds *RedisDataStructure) HKeys(key []byte) ([][]byte, error) {
	meta, err := rds.findMetadata(key, Hash)
	if err != nil {
		return nil, err
	}
	if meta.size == 0 {
		return nil, nil
	}
	prefixKey := &hashInternalKey{
		key:     key,
		version: meta.version,
		field:   nil,
	}
	prefix := prefixKey.encodePrefix()
	iterator := rds.db.NewIterator(bitcask.DefaultIteratorOptions)
	defer iterator.Close()

	keys := make([][]byte, 0, meta.size)
	for iterator.Seek(prefix); iterator.Valid(); iterator.Next() {
		if !bytes.HasPrefix(iterator.Key(), prefix) {
			break
		}
		// 解析处 field
		hk := decodeHashInternalKey(iterator.Key(), len(key))
		keys = append(keys, hk.field)
	}
	return keys, nil
}

func (rds *RedisDataStructure) HValues(key []byte) ([][]byte, error) {
	meta, err := rds.findMetadata(key, Hash)
	if err != nil {
		return nil, err
	}
	if meta.size == 0 {
		return nil, nil
	}
	prefixKey := &hashInternalKey{
		key:     key,
		version: meta.version,
		field:   nil,
	}
	prefix := prefixKey.encodePrefix()
	iterator := rds.db.NewIterator(bitcask.DefaultIteratorOptions)
	defer iterator.Close()
	values := make([][]byte, 0, meta.size)

	for iterator.Seek(prefix); iterator.Valid(); iterator.Next() {
		if !bytes.HasPrefix(iterator.Key(), prefix) {
			break
		}
		val, _ := iterator.Value()
		values = append(values, val)
	}
	return values, nil
}

func (rds *RedisDataStructure) HGetAll(key []byte) (map[string][]byte, error) {
	meta, err := rds.findMetadata(key, Hash)
	if err != nil {
		return nil, err
	}
	if meta.size == 0 {
		return nil, nil
	}
	prefixKey := &hashInternalKey{
		key:     key,
		version: meta.version,
		field:   nil,
	}
	prefix := prefixKey.encodePrefix()
	iterator := rds.db.NewIterator(bitcask.DefaultIteratorOptions)
	defer iterator.Close()
	result := make(map[string][]byte, meta.size)
	for iterator.Seek(prefix); iterator.Valid(); iterator.Next() {
		if !bytes.HasPrefix(iterator.Key(), prefix) {
			break
		}
		hk := decodeHashInternalKey(iterator.Key(), len(key))
		val, _ := iterator.Value()
		result[string(hk.field)] = val
	}
	return result, nil
}

// ++++++++++++++++++++++++++++ Set 数据结构 ++++++++++++++++++++++++++++
func (rds *RedisDataStructure) SAdd(key, member []byte) (bool, error) {
	meta, err := rds.findMetadata(key, Set)
	if err != nil {
		return false, err
	}
	sk := &setInternalKey{
		key:     key,
		version: meta.version,
		mumber:  member,
	}
	var ok bool
	if _, err = rds.db.Get(sk.enocde()); err == bitcask.ErrKeyNotFound {
		wb := rds.db.NewWriteBatch(bitcask.DefaultWriteBatchOptions)
		meta.size++
		_ = wb.Put(key, meta.encode())
		_ = wb.Put(sk.enocde(), nil)
		if err = wb.Commit(); err != nil {
			return false, err
		}
		ok = true
	}
	return ok, nil
}

func (rds *RedisDataStructure) SIsMember(key, member []byte) (bool, error) {
	meta, err := rds.findMetadata(key, Set)
	if err != nil {
		return false, err
	}
	if meta.size == 0 {
		return false, nil
	}
	sk := &setInternalKey{
		key:     key,
		version: meta.version,
		mumber:  member,
	}
	_, err = rds.db.Get(sk.enocde())
	if err != nil {
		if err == bitcask.ErrKeyNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (rds *RedisDataStructure) SRem(key, member []byte) (bool, error) {
	meta, err := rds.findMetadata(key, Set)
	if err != nil {
		return false, err
	}
	if meta.size == 0 {
		return false, nil
	}
	sk := &setInternalKey{
		key:     key,
		version: meta.version,
		mumber:  member,
	}
	if _, err = rds.db.Get(sk.enocde()); err == bitcask.ErrKeyNotFound {
		return false, nil
	}
	wb := rds.db.NewWriteBatch(bitcask.DefaultWriteBatchOptions)
	meta.size--
	_ = wb.Put(key, meta.encode())
	_ = wb.Delete(sk.enocde())
	if err = wb.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

// ++++++++++++++++++++++++++++ List 数据结构 ++++++++++++++++++++++++++++

func (rds *RedisDataStructure) LPush(key, element []byte) (uint32, error) {
	return rds.pushInner(key, element, true)
}

func (rds *RedisDataStructure) RPush(key, element []byte) (uint32, error) {
	return rds.pushInner(key, element, false)
}

func (rds *RedisDataStructure) LPop(key []byte) ([]byte, error) {
	return rds.popInner(key, true)
}

func (rds *RedisDataStructure) RPop(key []byte) ([]byte, error) {
	return rds.popInner(key, false)
}

func (rds *RedisDataStructure) LSize(key []byte) int64 {
	meta, err := rds.findMetadata(key, List)
	if err != nil {
		return 0
	}
	return int64(meta.size)
}

func (rds *RedisDataStructure) LAll(key []byte) (map[int64][]byte, error) {
	meta, err := rds.findMetadata(key, List)
	if err != nil {
		return nil, err
	}
	if meta.size == 0 {
		return make(map[int64][]byte), nil
	}
	result := make(map[int64][]byte, meta.size)
	var count int64
	for i := meta.head; i < meta.tail; i++ {
		lk := &listInternalKey{
			key:     key,
			version: meta.version,
			index:   i,
		}
		element, err := rds.db.Get(lk.encode())
		if err != nil {
			if err == bitcask.ErrKeyNotFound {
				return nil, fmt.Errorf("data inconsistency: expected element at index %d not found: %w", i, err)
			}
			return nil, err
		}
		result[count] = element
		count++
	}
	return result, nil
}

func (rds *RedisDataStructure) pushInner(key, element []byte, isLeft bool) (uint32, error) {
	meta, err := rds.findMetadata(key, List)
	if err != nil {
		return 0, err
	}
	lk := &listInternalKey{
		key:     key,
		version: meta.version,
	}
	if isLeft {
		lk.index = meta.head - 1
	} else {
		lk.index = meta.tail
	}
	// 更新元数据和数据部分
	wb := rds.db.NewWriteBatch(bitcask.DefaultWriteBatchOptions)
	meta.size++
	if isLeft {
		meta.head--
	} else {
		meta.tail++
	}
	_ = wb.Put(key, meta.encode())
	_ = wb.Put(lk.encode(), element)
	if err := wb.Commit(); err != nil {
		return 0, err
	}
	return meta.size, nil
}

func (rds *RedisDataStructure) popInner(key []byte, isLeft bool) ([]byte, error) {
	meta, err := rds.findMetadata(key, List)
	if err != nil {
		return nil, err
	}
	if meta.size == 0 {
		return nil, nil
	}
	lk := &listInternalKey{
		key:     key,
		version: meta.version,
	}
	if isLeft {
		lk.index = meta.head
	} else {
		lk.index = meta.tail - 1
	}
	element, err := rds.db.Get(lk.encode())
	if err != nil {
		return nil, err
	}

	// 更新元数据
	meta.size--
	if isLeft {
		meta.head++
	} else {
		meta.tail--
	}
	if err := rds.db.Delete(lk.encode()); err != nil {
		return nil, err
	}
	if err := rds.db.Put(key, meta.encode()); err != nil {
		return nil, err
	}
	return element, nil
}

// ++++++++++++++++++++++++++++ ZSet 数据结构 ++++++++++++++++++++++++++++

func (rds *RedisDataStructure) ZAdd(key []byte, score float64, member []byte) (bool, error) {
	meta, err := rds.findMetadata(key, ZSet)
	if err != nil {
		return false, err
	}

	// 构造数据部分的key
	zk := &zsetInternalKey{
		key:     key,
		member:  member,
		score:   score,
		version: meta.version,
	}
	// 查看是否已经存在
	var exist = true
	value, err := rds.db.Get(zk.encodeWithMember())
	if err != nil && err != bitcask.ErrKeyNotFound {
		return false, err
	}
	if err == bitcask.ErrKeyNotFound {
		exist = false
	}
	if exist {
		if score == utils.FloatFromBytes(value) {
			return false, nil
		}
	}

	// 更新元数据和数据
	wb := rds.db.NewWriteBatch(bitcask.DefaultWriteBatchOptions)
	if !exist {
		meta.size++
		wb.Put(key, meta.encode())
	}
	if exist {
		oldKey := &zsetInternalKey{
			key:     key,
			member:  member,
			version: meta.version,
			score:   utils.FloatFromBytes(value),
		}
		_ = wb.Delete(oldKey.encodeWithScore())
	}
	_ = wb.Put(zk.encodeWithMember(), utils.FLoatToBytes(score))
	_ = wb.Put(zk.encodeWithScore(), nil)
	if err := wb.Commit(); err != nil {
		return false, err
	}
	return !exist, nil

}

func (rds *RedisDataStructure) ZScore(key []byte, member []byte) (float64, error) {
	meta, err := rds.findMetadata(key, ZSet)
	if err != nil {
		return -1, err
	}
	if meta.size == 0 {
		return -1, nil
	}
	zk := &zsetInternalKey{
		key:     key,
		member:  member,
		version: meta.version,
	}
	value, err := rds.db.Get(zk.encodeWithMember())
	if err != nil {
		return -1, err
	}
	return utils.FloatFromBytes(value), nil
}

func (rds *RedisDataStructure) findMetadata(key []byte, dataType redisDataType) (*metadata, error) {
	metaBuf, err := rds.db.Get(key)
	if err != nil && err != bitcask.ErrKeyNotFound {
		return nil, err
	}
	var meta *metadata
	var exist = true
	if err == bitcask.ErrKeyNotFound {
		exist = false
	} else {
		meta = decodeMedtadata(metaBuf)
		// 判断数据类型
		if meta.dataType != dataType {
			return nil, ErrWrongTypeOperation
		}
		// 判断过期时间
		if meta.expire > 0 && time.Now().UnixNano() >= meta.expire {
			exist = false
		}
	}
	if !exist {
		meta = &metadata{
			dataType: dataType,
			expire:   0,
			version:  time.Now().UnixNano(),
			size:     0,
		}
		if dataType == List {
			meta.head = InitialListMark
			meta.tail = InitialListMark
		}
	}
	return meta, nil
}
