package fio

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func destory(name string) {
	if err := os.RemoveAll(name); err != nil {
		panic(err)
	}
}

func TestnewIOManager(t *testing.T) {
	path := filepath.Join("/tmp", "a.data")
	fio, err := NewFileIOManager(path)
	defer destory(path)

	assert.Nil(t, err)
	assert.NotNil(t, fio)
}

func TestWrite(t *testing.T) {
	path := filepath.Join("/tmp", "a.data")
	fio, _ := NewFileIOManager(path)
	defer destory(path)

	b := []byte("hello\n")
	n, err := fio.Write(b)
	assert.Nil(t, err)
	assert.Equal(t, 6, n)
}

func TestRead(t *testing.T) {
	path := filepath.Join("/tmp", "a.data")
	fio, _ := NewFileIOManager(path)
	defer destory(path)

	// 从文件初始位置写入
	n, err := fio.Write([]byte("key-a"))
	assert.Nil(t, err)
	assert.Equal(t, 5, n)
	// 从文件初始读
	b1 := make([]byte, 5)
	n, err = fio.Read(b1, 0)
	assert.Nil(t, err)
	assert.Equal(t, 5, n)
	// t.Log(string(b1))
	assert.Equal(t, 5, n)
	assert.Equal(t, []byte("key-a"), b1)

	n, err = fio.Write([]byte("key-b"))
	assert.Nil(t, err)
	assert.Equal(t, 5, n)
	b2 := make([]byte, 5)
	n, err = fio.Read(b2, 5)
	assert.Nil(t, err)
	assert.Equal(t, 5, n)
	// t.Log(string(b2))
	assert.Equal(t, 5, n)
	assert.Equal(t, []byte("key-b"), b2)
}

func TestSync(t *testing.T) {
	path := filepath.Join("/tmp", "a.data")
	fio, _ := NewFileIOManager(path)
	defer destory(path)

	n, err := fio.Write([]byte("hello"))
	assert.Nil(t, err)
	assert.Equal(t, 5, n)

	err = fio.Sync()
	assert.Nil(t, err)
}

func TestClose(t *testing.T) {
	path := filepath.Join("/tmp", "a.data")
	fio, _ := NewFileIOManager(path)
	defer destory(path)

	n, err := fio.Write([]byte("hello"))
	assert.Nil(t, err)
	assert.Equal(t, 5, n)

	err = fio.Close()
	assert.Nil(t, err)
}
