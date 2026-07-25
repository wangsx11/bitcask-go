package fio

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newTestFileIO(t *testing.T) *FileIO {
	t.Helper()

	ioManager, err := NewFileIOManager(filepath.Join(t.TempDir(), "test.data"))
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, ioManager.Close())
	})
	return ioManager
}

func TestNewIOManager(t *testing.T) {
	ioManager := newTestFileIO(t)
	assert.NotNil(t, ioManager)
}

func TestWrite(t *testing.T) {
	ioManager := newTestFileIO(t)

	n, err := ioManager.Write([]byte("hello\n"))
	assert.NoError(t, err)
	assert.Equal(t, 6, n)
}

func TestRead(t *testing.T) {
	ioManager := newTestFileIO(t)

	n, err := ioManager.Write([]byte("key-a"))
	assert.NoError(t, err)
	assert.Equal(t, 5, n)

	first := make([]byte, 5)
	n, err = ioManager.Read(first, 0)
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, []byte("key-a"), first)

	n, err = ioManager.Write([]byte("key-b"))
	assert.NoError(t, err)
	assert.Equal(t, 5, n)

	second := make([]byte, 5)
	n, err = ioManager.Read(second, 5)
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, []byte("key-b"), second)
}

func TestSync(t *testing.T) {
	ioManager := newTestFileIO(t)

	_, err := ioManager.Write([]byte("hello"))
	assert.NoError(t, err)
	assert.NoError(t, ioManager.Sync())
}

func TestClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.data")
	ioManager, err := NewFileIOManager(path)
	assert.NoError(t, err)

	_, err = ioManager.Write([]byte("hello"))
	assert.NoError(t, err)
	assert.NoError(t, ioManager.Close())
}
