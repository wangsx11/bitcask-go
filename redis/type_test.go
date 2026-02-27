package redis

import (
	bitcask "bitcask-go"
	"bitcask-go/utils"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRedisDataStructure_Get(t *testing.T) {
	opts := bitcask.DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-redis-get")
	opts.DirPath = dir
	rds, err := NewRedisDataStructure(opts)
	assert.Nil(t, err)

	err = rds.Set(utils.GetTestKey(1), 0, utils.RandomValue(100))
	assert.Nil(t, err)
	err = rds.Set(utils.GetTestKey(2), time.Second*2, utils.RandomValue(200))
	assert.Nil(t, err)
	value1, err := rds.Get(utils.GetTestKey(1))
	assert.Nil(t, err)
	assert.NotNil(t, value1)

	value2, err := rds.Get(utils.GetTestKey(2))
	assert.Nil(t, err)
	assert.NotNil(t, value2)

	_, err = rds.Get(utils.GetTestKey(3))
	assert.Equal(t, bitcask.ErrKeyNotFound, err)

	time.Sleep(time.Second * 3)
	_, err = rds.Get(utils.GetTestKey(2))
	assert.Equal(t, ErrKeyExpired, err)
}

func TestRedisDataStructure_Del_Type(t *testing.T) {
	opts := bitcask.DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-redis-del")
	opts.DirPath = dir
	rds, err := NewRedisDataStructure(opts)
	assert.Nil(t, err)

	//del

	err = rds.Delete(utils.GetTestKey(1))
	assert.Nil(t, err)
	err = rds.Set(utils.GetTestKey(1), 0, utils.RandomValue(100))
	assert.Nil(t, err)

	err = rds.Delete(utils.GetTestKey(1))
	assert.Nil(t, err)

	value1, err := rds.Get(utils.GetTestKey(1))
	assert.Nil(t, value1)
	assert.Equal(t, bitcask.ErrKeyNotFound, err)

	// type
	err = rds.Set(utils.GetTestKey(1), 0, utils.RandomValue(100))
	assert.Nil(t, err)
	typ, err := rds.Type(utils.GetTestKey(1))
	assert.Nil(t, err)
	assert.Equal(t, String, typ)

	err = rds.Delete(utils.GetTestKey(1))
	assert.Nil(t, err)
	typ, err = rds.Type(utils.GetTestKey(1))
	assert.Equal(t, bitcask.ErrKeyNotFound, err)
	assert.Equal(t, uint8(0), typ)
}

func TestRedisDataStructure_HGet(t *testing.T) {
	opts := bitcask.DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-redis-hget")
	opts.DirPath = dir
	rds, err := NewRedisDataStructure(opts)
	assert.Nil(t, err)

	ok1, err := rds.HSet(utils.GetTestKey(1), []byte("field1"), utils.RandomValue(100))
	assert.Nil(t, err)
	t.Log(ok1)
	ok2, err := rds.HSet(utils.GetTestKey(1), []byte("field1"), utils.RandomValue(100))
	assert.Nil(t, err)
	t.Log(ok2)
	ok3, err := rds.HSet(utils.GetTestKey(1), []byte("field2"), utils.RandomValue(100))
	assert.Nil(t, err)
	t.Log(ok3)

	val1, err := rds.HGet(utils.GetTestKey(1), []byte("field1"))
	assert.Nil(t, err)
	t.Log(string(val1))

	val2, err := rds.HGet(utils.GetTestKey(1), []byte("field-not-exist"))
	assert.Equal(t, err, bitcask.ErrKeyNotFound)
	t.Log(string(val2), err)

}

func TestRedisDataStructure_HDel(t *testing.T) {
	opts := bitcask.DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-redis-hdel")
	opts.DirPath = dir
	rds, err := NewRedisDataStructure(opts)
	assert.Nil(t, err)
	del1, err := rds.HDel(utils.GetTestKey(200), nil)
	assert.Nil(t, err)
	assert.Equal(t, del1, false)

	ok1, err := rds.HSet(utils.GetTestKey(1), []byte("field1"), utils.RandomValue(100))
	assert.Nil(t, err)
	t.Log(ok1)
	ok2, err := rds.HSet(utils.GetTestKey(1), []byte("field1"), utils.RandomValue(100))
	assert.Nil(t, err)
	t.Log(ok2)
	ok3, err := rds.HSet(utils.GetTestKey(1), []byte("field2"), utils.RandomValue(100))
	assert.Nil(t, err)
	t.Log(ok3)

	del2, err := rds.HDel(utils.GetTestKey(1), []byte("field1"))
	t.Log(del2, err)
}

func TestRedisDataStructure_HKeys(t *testing.T) {
	opts := bitcask.DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-redis-hkeys")
	opts.DirPath = dir
	rds, err := NewRedisDataStructure(opts)
	assert.Nil(t, err)

	// 测试空的 hash
	keys1, err := rds.HKeys(utils.GetTestKey(100))
	assert.Nil(t, err)
	assert.Nil(t, keys1)

	// 设置一些 hash 数据
	ok1, err := rds.HSet(utils.GetTestKey(1), []byte("field1"), utils.RandomValue(100))
	assert.Nil(t, err)
	assert.True(t, ok1)

	ok2, err := rds.HSet(utils.GetTestKey(1), []byte("field2"), utils.RandomValue(100))
	assert.Nil(t, err)
	assert.True(t, ok2)

	ok3, err := rds.HSet(utils.GetTestKey(1), []byte("field3"), utils.RandomValue(100))
	assert.Nil(t, err)
	assert.True(t, ok3)

	// 测试获取所有 keys
	keys2, err := rds.HKeys(utils.GetTestKey(1))
	assert.Nil(t, err)
	assert.Equal(t, 3, len(keys2))
	keyStrings := make([]string, len(keys2))
	for i, key := range keys2 {
		keyStrings[i] = string(key)
	}
	t.Log("HKeys result:", keyStrings)
}

func TestRedisDataStructure_HValues(t *testing.T) {
	opts := bitcask.DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-redis-hvalues")
	opts.DirPath = dir
	rds, err := NewRedisDataStructure(opts)
	assert.Nil(t, err)

	// 测试空的 hash
	values1, err := rds.HValues(utils.GetTestKey(200))
	assert.Nil(t, err)
	assert.Nil(t, values1)

	// 设置一些 hash 数据
	value1 := utils.RandomValue(100)
	value2 := utils.RandomValue(100)
	value3 := utils.RandomValue(100)

	ok1, err := rds.HSet(utils.GetTestKey(2), []byte("field1"), value1)
	assert.Nil(t, err)
	assert.True(t, ok1)

	ok2, err := rds.HSet(utils.GetTestKey(2), []byte("field2"), value2)
	assert.Nil(t, err)
	assert.True(t, ok2)

	ok3, err := rds.HSet(utils.GetTestKey(2), []byte("field3"), value3)
	assert.Nil(t, err)
	assert.True(t, ok3)

	// 测试获取所有 values
	values2, err := rds.HValues(utils.GetTestKey(2))
	assert.Nil(t, err)
	assert.Equal(t, 3, len(values2))
	assert.Equal(t, value1, values2[0])
	assert.Equal(t, value2, values2[1])
	assert.Equal(t, value3, values2[2])
}

func TestRedisDataStructure_HGetAll(t *testing.T) {
	opts := bitcask.DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-redis-hgetall")
	opts.DirPath = dir
	rds, err := NewRedisDataStructure(opts)
	assert.Nil(t, err)

	// 测试空的 hash
	result1, err := rds.HGetAll(utils.GetTestKey(300))
	assert.Nil(t, err)
	assert.Equal(t, 0, len(result1))

	// 设置一些 hash 数据
	value1 := []byte("test_value_1")
	value2 := []byte("test_value_2")
	value3 := []byte("test_value_3")

	ok1, err := rds.HSet(utils.GetTestKey(3), []byte("name"), value1)
	assert.Nil(t, err)
	assert.True(t, ok1)

	ok2, err := rds.HSet(utils.GetTestKey(3), []byte("age"), value2)
	assert.Nil(t, err)
	assert.True(t, ok2)

	ok3, err := rds.HSet(utils.GetTestKey(3), []byte("email"), value3)
	assert.Nil(t, err)
	assert.True(t, ok3)

	// 测试获取所有 field-value 对
	result2, err := rds.HGetAll(utils.GetTestKey(3))
	assert.Nil(t, err)
	assert.Equal(t, 3, len(result2))

	// 验证具体的值
	assert.Equal(t, value1, result2["name"])
	assert.Equal(t, value2, result2["age"])
	assert.Equal(t, value3, result2["email"])
}

func TestRedisDataStructure_SIsMember(t *testing.T) {
	opts := bitcask.DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-redis-sismember")
	opts.DirPath = dir
	rds, err := NewRedisDataStructure(opts)
	assert.Nil(t, err)

	ok1, err := rds.SAdd(utils.GetTestKey(1), []byte("member1"))
	assert.Nil(t, err)
	assert.Equal(t, true, ok1)
	ok2, err := rds.SAdd(utils.GetTestKey(1), []byte("member1"))
	assert.Nil(t, err)
	assert.Equal(t, false, ok2)

	ok3, err := rds.SAdd(utils.GetTestKey(1), []byte("member2"))
	assert.Nil(t, err)
	assert.Equal(t, true, ok3)

	f1, err := rds.SIsMember(utils.GetTestKey(2), nil)
	assert.Equal(t, false, f1)
	assert.Nil(t, err)

	f2, err := rds.SIsMember(utils.GetTestKey(1), []byte("member1"))
	assert.Equal(t, true, f2)
	assert.Nil(t, err)

	f3, err := rds.SIsMember(utils.GetTestKey(1), []byte("member-not-exist"))
	assert.Equal(t, false, f3)
	assert.Nil(t, err)
}

func TestRedisDataStructure_SRem(t *testing.T) {
	opts := bitcask.DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-redis-sismember")
	opts.DirPath = dir
	rds, err := NewRedisDataStructure(opts)
	assert.Nil(t, err)

	ok1, err := rds.SAdd(utils.GetTestKey(1), []byte("member1"))
	assert.Nil(t, err)
	assert.Equal(t, true, ok1)

	ok2, err := rds.SAdd(utils.GetTestKey(1), []byte("member2"))
	assert.Nil(t, err)
	assert.Equal(t, true, ok2)

	f1, err := rds.SRem(utils.GetTestKey(2), nil)
	assert.Equal(t, false, f1)
	assert.Nil(t, err)

	f2, err := rds.SRem(utils.GetTestKey(1), []byte("member1"))
	assert.Equal(t, true, f2)
	assert.Nil(t, err)

	f3, err := rds.SRem(utils.GetTestKey(1), []byte("member-not-exist"))
	assert.Equal(t, false, f3)
	assert.Nil(t, err)
}

func TestRedisDataStructure_List(t *testing.T) {
	opts := bitcask.DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-redis-list")
	opts.DirPath = dir
	rds, err := NewRedisDataStructure(opts)
	assert.Nil(t, err)
	res, err := rds.LPush(utils.GetTestKey(1), []byte("member-1"))
	t.Log(res, err)
	res, err = rds.LPush(utils.GetTestKey(1), []byte("member-2"))
	t.Log(res, err)
	res, err = rds.LPush(utils.GetTestKey(1), []byte("member-3"))
	t.Log(res, err)
	res, err = rds.LPush(utils.GetTestKey(1), []byte("member-4"))
	t.Log(res, err)

	res, err = rds.RPush(utils.GetTestKey(1), []byte("member-5"))
	t.Log(res, err)
	res, err = rds.RPush(utils.GetTestKey(1), []byte("member-6"))
	t.Log(res, err)
	res, err = rds.RPush(utils.GetTestKey(1), []byte("member-7"))
	t.Log(res, err)

	f1, err := rds.LPop(utils.GetTestKey(1))
	assert.Nil(t, err)
	assert.Equal(t, "member-4", string(f1))

	f2, err := rds.RPop(utils.GetTestKey(1))
	assert.Nil(t, err)
	assert.Equal(t, "member-7", string(f2))
	f3, err := rds.RPop(utils.GetTestKey(1))
	assert.Nil(t, err)
	assert.Equal(t, "member-6", string(f3))
	size := rds.LSize(utils.GetTestKey(1))
	assert.Equal(t, int64(4), size)

	res, err = rds.LPush(utils.GetTestKey(1), []byte("member-100"))
	t.Log(res, err)
	f, err := rds.LPop(utils.GetTestKey(1))
	assert.Nil(t, err)
	assert.Equal(t, "member-100", string(f))

	res, err = rds.LPush(utils.GetTestKey(1), []byte("member-100"))
	t.Log(res, err)
	f, err = rds.LPop(utils.GetTestKey(1))
	assert.Nil(t, err)
	assert.Equal(t, "member-100", string(f))

	res, err = rds.LPush(utils.GetTestKey(1), []byte("member-100"))
	t.Log(res, err)
	f, err = rds.LPop(utils.GetTestKey(1))
	assert.Nil(t, err)
	assert.Equal(t, "member-100", string(f))
}

func TestRedisDataStructure_LAll(t *testing.T) {
	opts := bitcask.DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-redis-lall")
	opts.DirPath = dir
	rds, err := NewRedisDataStructure(opts)
	assert.Nil(t, err)

	// 测试空列表
	result1, err := rds.LAll(utils.GetTestKey(1))
	assert.Nil(t, err)
	assert.Equal(t, 0, len(result1))
	t.Log("Empty list result:", result1)

	// 添加一些元素
	res, err := rds.LPush(utils.GetTestKey(1), []byte("left-1"))
	assert.Nil(t, err)
	t.Log("LPush left-1, size:", res)

	res, err = rds.LPush(utils.GetTestKey(1), []byte("left-2"))
	assert.Nil(t, err)
	t.Log("LPush left-2, size:", res)

	res, err = rds.RPush(utils.GetTestKey(1), []byte("right-1"))
	assert.Nil(t, err)
	t.Log("RPush right-1, size:", res)

	res, err = rds.RPush(utils.GetTestKey(1), []byte("right-2"))
	assert.Nil(t, err)
	t.Log("RPush right-2, size:", res)

	// 测试获取所有元素
	result2, err := rds.LAll(utils.GetTestKey(1))
	assert.Nil(t, err)
	assert.Equal(t, 4, len(result2))

	// 转换为字符串输出
	result2Str := make(map[string]string)
	for index, element := range result2 {
		result2Str[fmt.Sprintf("%d", index)] = string(element)
	}
	t.Log("LAll result:", result2Str)

	// 简单验证包含预期的元素
	foundElements := make(map[string]bool)
	for _, element := range result2 {
		foundElements[string(element)] = true
	}

	assert.True(t, foundElements["left-1"])
	assert.True(t, foundElements["left-2"])
	assert.True(t, foundElements["right-1"])
	assert.True(t, foundElements["right-2"])

	// 弹出一个元素后再测试
	popped, err := rds.LPop(utils.GetTestKey(1))
	assert.Nil(t, err)
	t.Log("Popped:", string(popped))

	result3, err := rds.LAll(utils.GetTestKey(1))
	assert.Nil(t, err)
	assert.Equal(t, 3, len(result3))

	// 转换为字符串输出
	result3Str := make(map[string]string)
	for index, element := range result3 {
		result3Str[fmt.Sprintf("%d", index)] = string(element)
	}
	t.Log("After LPop, LAll result:", result3Str)
}

func TestRedisDataStructure_ZScore(t *testing.T) {
	opts := bitcask.DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-redis-lall")
	opts.DirPath = dir
	rds, err := NewRedisDataStructure(opts)
	assert.Nil(t, err)
	ok, err := rds.ZAdd(utils.GetTestKey(1), 1.0, []byte("member-1"))
	assert.Nil(t, err)
	assert.Equal(t, true, ok)
	ok, err = rds.ZAdd(utils.GetTestKey(1), 2.0, []byte("member-1"))
	assert.Nil(t, err)
	assert.Equal(t, false, ok)
	ok, err = rds.ZAdd(utils.GetTestKey(1), 3.0, []byte("member-3"))
	assert.Nil(t, err)
	assert.Equal(t, true, ok)

	score, err := rds.ZScore(utils.GetTestKey(1), []byte("member-1"))
	assert.Nil(t, err)
	assert.Equal(t, 2.0, score)

	score, err = rds.ZScore(utils.GetTestKey(1), []byte("member-3"))
	assert.Nil(t, err)
	assert.Equal(t, 3.0, score)
}
