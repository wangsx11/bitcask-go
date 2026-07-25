package fio

import (
	"io"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMMapRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mmap.data")

	emptyMMap, err := NewMMapIOManager(path)
	assert.NoError(t, err)
	buffer := make([]byte, 10)
	n, err := emptyMMap.Read(buffer, 0)
	assert.Equal(t, 0, n)
	assert.ErrorIs(t, err, io.EOF)
	assert.NoError(t, emptyMMap.Close())

	fileIO, err := NewFileIOManager(path)
	assert.NoError(t, err)
	_, err = fileIO.Write([]byte("aabbcc"))
	assert.NoError(t, err)
	assert.NoError(t, fileIO.Close())

	mmapIO, err := NewMMapIOManager(path)
	assert.NoError(t, err)
	defer func() { assert.NoError(t, mmapIO.Close()) }()

	size, err := mmapIO.Size()
	assert.NoError(t, err)
	assert.Equal(t, int64(6), size)

	value := make([]byte, 2)
	n, err = mmapIO.Read(value, 0)
	assert.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.Equal(t, "aa", string(value))
}

func TestMMapIsReadOnly(t *testing.T) {
	fileName := filepath.Join(t.TempDir(), "readonly.data")
	mmapIO, err := NewMMapIOManager(fileName)
	assert.NoError(t, err)
	defer func() { assert.NoError(t, mmapIO.Close()) }()

	_, err = mmapIO.Write([]byte("value"))
	assert.ErrorIs(t, err, ErrReadOnly)
	assert.ErrorIs(t, mmapIO.Sync(), ErrReadOnly)
}

func TestNewIOManagerUnsupportedType(t *testing.T) {
	manager, err := NewIOManager(filepath.Join(t.TempDir(), "data"), 99)
	assert.Nil(t, manager)
	assert.ErrorIs(t, err, ErrUnsupportedIOType)
}
