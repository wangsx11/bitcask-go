package bitcask_go

import "errors"

var (
	ErrKeyIsEmpty             = errors.New("the key is empty")
	ErrIndexUpdateFailed      = errors.New("failed to updata index")
	ErrKeyNotFound            = errors.New("key not found in database")
	ErrDataFileNotFound       = errors.New("data file is not found")
	ErrDataDirectoryCorrupted = errors.New("the database directory mabey corrupted")
	ErrExceedMaxBatchNum      = errors.New("exceed the max batch num")
	ErrMergIsProcess          = errors.New("merge is in process, try again later")
	ErrDatabaseIsUsing        = errors.New("the database is used by another process")
	ErrMergeRatioUnreached    = errors.New("the merge ratio do not reached")
	ErrNoEnoughSpaceMerge     = errors.New("no enough space to merge")
)
