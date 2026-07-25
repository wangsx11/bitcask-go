package bitcask_go

import "errors"

var (
	ErrClosed            = errors.New("database is closed")
	ErrCorrupted         = errors.New("database is corrupted")
	ErrUnsupported       = errors.New("unsupported operation or type")
	ErrInvalidOptions    = errors.New("invalid options")
	ErrKeyIsEmpty        = errors.New("the key is empty")
	ErrIndexUpdateFailed = errors.New("failed to update index")
	ErrKeyNotFound       = errors.New("key not found in database")
	ErrDataFileNotFound  = errors.New("data file is not found")
	// Deprecated: use ErrCorrupted.
	ErrDataDirectoryCorrupted = ErrCorrupted
	ErrExceedMaxBatchNum      = errors.New("exceed the max batch num")
	ErrMergeIsProcessing      = errors.New("merge is in process, try again later")
	// Deprecated: use ErrMergeIsProcessing.
	ErrMergIsProcess       = ErrMergeIsProcessing
	ErrDatabaseIsUsing     = errors.New("the database is used by another process")
	ErrMergeRatioUnreached = errors.New("merge ratio is not reached")
	ErrNoEnoughSpaceMerge  = errors.New("not enough space to merge")
)
