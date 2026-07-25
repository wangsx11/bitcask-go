package bitcask_go

import (
	"bitcask-go/data"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultOptionsAreIndependent(t *testing.T) {
	first := DefaultOptions()
	second := DefaultOptions()
	first.DirPath = t.TempDir()
	first.SyncWrites = true

	assert.NotEqual(t, first.DirPath, second.DirPath)
	assert.False(t, second.SyncWrites)
}

func TestOpenCopiesOptions(t *testing.T) {
	originalDir := t.TempDir()
	otherDir := t.TempDir()
	opts := DefaultOptions()
	opts.DirPath = originalDir

	db, err := Open(&opts)
	assert.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, db.Close()) })

	opts.DirPath = otherDir
	opts.SyncWrites = true
	assert.NoError(t, db.Put([]byte("key"), []byte("value")))

	assert.Equal(t, originalDir, db.options.DirPath)
	assert.False(t, db.options.SyncWrites)
	_, err = os.Stat(filepath.Join(originalDir, "000000000.data"))
	assert.NoError(t, err)
}

func TestMergeCopiesOptions(t *testing.T) {
	opts := DefaultOptions()
	opts.DirPath = t.TempDir()
	opts.SyncWrites = true
	opts.DataFileMergeRatio = 0
	db, err := Open(&opts)
	assert.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
		_ = os.RemoveAll(db.getMergePath())
	})

	assert.NoError(t, db.Put([]byte("key"), []byte("value")))
	assert.NoError(t, db.Merge())
	assert.Equal(t, opts.DirPath, db.options.DirPath)
	assert.True(t, opts.SyncWrites)
	assert.True(t, db.options.SyncWrites)
}

func TestOpenRejectsInvalidOptions(t *testing.T) {
	valid := DefaultOptions()
	valid.DirPath = t.TempDir()

	tests := []struct {
		name   string
		opts   *Options
		target error
	}{
		{name: "nil", opts: nil, target: ErrInvalidOptions},
		{name: "empty directory", opts: func() *Options { o := valid; o.DirPath = ""; return &o }(), target: ErrInvalidOptions},
		{name: "zero file size", opts: func() *Options { o := valid; o.DataFileSize = 0; return &o }(), target: ErrInvalidOptions},
		{name: "negative merge ratio", opts: func() *Options { o := valid; o.DataFileMergeRatio = -0.1; return &o }(), target: ErrInvalidOptions},
		{name: "large merge ratio", opts: func() *Options { o := valid; o.DataFileMergeRatio = 1.1; return &o }(), target: ErrInvalidOptions},
		{name: "nan merge ratio", opts: func() *Options { o := valid; o.DataFileMergeRatio = float32(math.NaN()); return &o }(), target: ErrInvalidOptions},
		{name: "unsupported index", opts: func() *Options { o := valid; o.IndexerType = 99; return &o }(), target: ErrUnsupported},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := Open(tt.opts)
			assert.Nil(t, db)
			assert.ErrorIs(t, err, tt.target)
		})
	}
}

func TestOpenRejectsFileAsDirectory(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "database-file")
	assert.NoError(t, err)
	assert.NoError(t, file.Close())
	opts := DefaultOptions()
	opts.DirPath = file.Name()

	db, err := Open(&opts)
	assert.Nil(t, db)
	assert.ErrorIs(t, err, ErrInvalidOptions)
}

func TestOpenFailureReleasesResourcesAndLock(t *testing.T) {
	dir := t.TempDir()
	dataFile := filepath.Join(dir, "000000000.data")
	assert.NoError(t, os.WriteFile(dataFile, []byte{1, 2, 3, 4, 0, 0}, 0o644))
	opts := DefaultOptions()
	opts.DirPath = dir

	db, err := Open(&opts)
	assert.Nil(t, db)
	assert.ErrorIs(t, err, ErrCorrupted)
	_, statErr := os.Stat(filepath.Join(dir, "seq-no"))
	assert.ErrorIs(t, statErr, os.ErrNotExist)

	assert.NoError(t, os.Truncate(dataFile, 0))
	db, err = Open(&opts)
	assert.NoError(t, err)
	assert.NoError(t, db.Close())
}

func TestStatReturnsDirectoryError(t *testing.T) {
	dir := t.TempDir()
	movedDir := dir + "-moved"
	opts := DefaultOptions()
	opts.DirPath = dir
	db, err := Open(&opts)
	assert.NoError(t, err)
	assert.NoError(t, os.Rename(dir, movedDir))

	_, err = db.Stat()
	assert.Error(t, err)

	assert.NoError(t, os.Rename(movedDir, dir))
	assert.NoError(t, db.Close())
}

func TestOpenBPlusTreeFailureReleasesLock(t *testing.T) {
	dir := t.TempDir()
	assert.NoError(t, os.Mkdir(filepath.Join(dir, "bptree-index"), 0o755))
	opts := DefaultOptions()
	opts.DirPath = dir
	opts.IndexerType = BPlusTree

	db, err := Open(&opts)
	assert.Nil(t, db)
	assert.Error(t, err)

	opts.IndexerType = ART
	db, err = Open(&opts)
	assert.NoError(t, err)
	assert.NoError(t, db.Close())
}

func TestBPlusTreeWriteBatchInEmptyDirectory(t *testing.T) {
	opts := DefaultOptions()
	opts.DirPath = t.TempDir()
	opts.IndexerType = BPlusTree
	db, err := Open(&opts)
	assert.NoError(t, err)

	batch, err := db.NewWriteBatch(DefaultWriteBatchOptions)
	assert.NoError(t, err)
	assert.NoError(t, batch.Put([]byte("key"), []byte("value")))
	assert.NoError(t, batch.Commit())
	assert.NoError(t, db.Close())

	db, err = Open(&opts)
	assert.NoError(t, err)
	_, err = db.NewWriteBatch(DefaultWriteBatchOptions)
	assert.NoError(t, err)
	assert.NoError(t, db.Close())
}

func TestOpenPermissionFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permissions differ on Windows")
	}
	dir := t.TempDir()
	assert.NoError(t, os.Chmod(dir, 0))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	opts := DefaultOptions()
	opts.DirPath = dir

	db, err := Open(&opts)
	if err == nil {
		_ = db.Close()
		t.Skip("current user can access a mode 000 directory")
	}
	assert.Nil(t, db)
}

func TestCloseIsIdempotentAndReleasesLock(t *testing.T) {
	opts := testOptions(t)
	db, err := Open(opts)
	assert.NoError(t, err)

	assert.NoError(t, db.Close())
	assert.NoError(t, db.Close())

	reopened, err := Open(opts)
	assert.NoError(t, err)
	assert.NoError(t, reopened.Close())
}

func TestCloseDoesNotCreateSeqNoForMemoryIndex(t *testing.T) {
	opts := testOptions(t)
	db, err := Open(opts)
	assert.NoError(t, err)
	assert.NoError(t, db.Put([]byte("key"), []byte("value")))
	assert.NoError(t, db.Close())

	_, err = os.Stat(filepath.Join(opts.DirPath, data.SeqNoFileName))
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestOperationsAfterCloseReturnErrClosed(t *testing.T) {
	opts := testOptions(t)
	db, err := Open(opts)
	assert.NoError(t, err)
	assert.NoError(t, db.Put([]byte("key"), []byte("value")))
	iterator, err := db.NewIterator(DefaultIteratorOptions)
	assert.NoError(t, err)
	batch, err := db.NewWriteBatch(DefaultWriteBatchOptions)
	assert.NoError(t, err)
	assert.NoError(t, db.Close())

	assert.ErrorIs(t, db.Put([]byte("key"), []byte("value")), ErrClosed)
	_, err = db.Get([]byte("key"))
	assert.ErrorIs(t, err, ErrClosed)
	assert.ErrorIs(t, db.Delete([]byte("key")), ErrClosed)
	assert.ErrorIs(t, db.Sync(), ErrClosed)
	_, err = db.Stat()
	assert.ErrorIs(t, err, ErrClosed)
	_, err = db.ListKeys()
	assert.ErrorIs(t, err, ErrClosed)
	assert.ErrorIs(t, db.Fold(func([]byte, []byte) bool { return true }), ErrClosed)
	assert.ErrorIs(t, db.Backup(t.TempDir()), ErrClosed)
	_, err = db.NewIterator(DefaultIteratorOptions)
	assert.ErrorIs(t, err, ErrClosed)
	_, err = db.NewWriteBatch(DefaultWriteBatchOptions)
	assert.ErrorIs(t, err, ErrClosed)
	assert.ErrorIs(t, db.Merge(), ErrClosed)
	assert.ErrorIs(t, batch.Put([]byte("key"), []byte("value")), ErrClosed)
	assert.ErrorIs(t, batch.Delete([]byte("key")), ErrClosed)
	assert.ErrorIs(t, batch.Commit(), ErrClosed)
	_, err = iterator.Value()
	assert.ErrorIs(t, err, ErrClosed)
	assert.NoError(t, iterator.Close())
}

func TestNewWriteBatchRejectsZeroLimit(t *testing.T) {
	opts := testOptions(t)
	db, err := Open(opts)
	assert.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, db.Close()) })

	_, err = db.NewWriteBatch(WriteBatchOptions{})
	assert.ErrorIs(t, err, ErrInvalidOptions)
}
