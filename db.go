package bitcask_go

import (
	"bitcask-go/data"
	"bitcask-go/fio"
	"bitcask-go/index"
	"bitcask-go/utils"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/gofrs/flock"

	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const (
	seqNoKey     = "seq.no"
	fileLockName = "flock"
)

// DB bitcask 存储引擎实例
type DB struct {
	mu              *sync.RWMutex
	closeMu         sync.Mutex
	mergeWG         sync.WaitGroup
	options         Options
	fileIds         []int                     // 列表只能用于加载索引时使用
	activeFile      *data.DataFile            // 当前活跃数据文件，可以用于写入
	olderFiles      map[uint32]*data.DataFile // 旧的数据文件，只能用于读
	index           index.Indexer             // 内存索引
	seqNo           uint64                    // 事务序列号，全局递增
	isMerging       bool                      // 是否增在进行 merge
	seqNoFileExists bool                      // 是否存在 seq.no 文件
	isInitial       bool                      // 是否是第一次打开数据库
	fileLock        *flock.Flock              // 文件锁， 保证多进程之间互斥
	bytesWrite      uint                      // 累计写了多少个字节
	reclaimSize     int64                     // 表示有多少字节可以进行回收
	closed          bool
}

// 存储引擎统计信息
type Stat struct {
	KeyNum          uint  // key 的总数
	DataFileNum     uint  // 数据文件总数
	ReclaimableSize int64 // 可回收字节数， 以字节为单位
	DiskSize        int64 // 数据目录所占磁盘空间
}

// Open 打开 bitcask 存储引擎
func Open(options *Options) (result *DB, retErr error) {
	defer func() {
		if errors.Is(retErr, data.ErrInvalidCRC) && !errors.Is(retErr, ErrCorrupted) {
			retErr = fmt.Errorf("%w: %w", ErrCorrupted, retErr)
		}
	}()
	if err := checkOptions(options); err != nil {
		return nil, err
	}
	opts := *options
	var isInitial bool

	// 判断目录是否存在 如果不存在，则创建这个目录
	if info, err := os.Stat(opts.DirPath); os.IsNotExist(err) {
		if err := os.MkdirAll(opts.DirPath, os.ModePerm); err != nil {
			return nil, err
		}
		// 创建成功，说明完成首次启动DB实例
		isInitial = true
	} else if err != nil {
		return nil, err
	} else if !info.IsDir() {
		return nil, fmt.Errorf("%w: database path is not a directory", ErrInvalidOptions)
	}

	// 判断当前数据目录是否正在使用
	fileLock := flock.New(filepath.Join(opts.DirPath, fileLockName))
	hold, err := fileLock.TryLock()
	if err != nil {
		return nil, err
	}
	if !hold {
		return nil, ErrDatabaseIsUsing
	}

	entrys, err := os.ReadDir(opts.DirPath)
	if err != nil {
		return nil, errors.Join(err, fileLock.Unlock())
	}
	if !isInitial {
		isInitial = true
		for _, entry := range entrys {
			if strings.HasSuffix(entry.Name(), data.DataFileNameSuffix) || entry.Name() == data.SeqNoFileName {
				isInitial = false
				break
			}
		}
	}

	// 初始化 DB 实例结构体
	idx, err := index.NewIndexer(opts.IndexerType, opts.DirPath, opts.SyncWrites)
	if err != nil {
		unlockErr := fileLock.Unlock()
		if errors.Is(err, index.ErrUnsupportedIndexType) {
			return nil, errors.Join(fmt.Errorf("%w: %v", ErrUnsupported, err), unlockErr)
		}
		return nil, errors.Join(err, unlockErr)
	}
	db := &DB{
		mu:         new(sync.RWMutex),
		options:    opts,
		olderFiles: make(map[uint32]*data.DataFile),
		index:      idx,
		isInitial:  isInitial,
		fileLock:   fileLock,
	}
	opened := false
	defer func() {
		if !opened {
			retErr = errors.Join(retErr, db.cleanupOpenFailure())
		}
	}()

	// 加载 merge 数据目录
	if err := db.LoadMergeFiles(); err != nil {
		return nil, err
	}

	// 加载数据文件
	if err := db.loadDataFiles(); err != nil {
		return nil, err
	}

	// B+树索引不需要从数据文件中加载索引
	if opts.IndexerType != BPlusTree {
		// 从 hint 索引文件中加载索引
		if err := db.loadIndexFromHintFile(); err != nil {
			return nil, err
		}

		// 从数据文件中加载索引
		if err := db.loadIndexFromDataFiles(); err != nil {
			return nil, err
		}

		// 重置 IO 类型 为标准文件IO
		if db.options.MMapAtStartup {
			if err := db.resetIoType(); err != nil {
				return nil, err
			}
		}
	}

	// 取出当前事务序列号
	if opts.IndexerType == BPlusTree {
		if err := db.loadSeqNo(); err != nil {
			return nil, err
		}
		if db.activeFile != nil {
			size, err := db.activeFile.IoManager.Size()
			if err != nil {
				return nil, err
			}
			db.activeFile.WriteOff = size
		}
	}

	opened = true
	return db, nil
}

// Close 关闭数据库
func (db *DB) Close() error {
	db.closeMu.Lock()
	defer db.closeMu.Unlock()

	db.mu.Lock()
	if db.closed {
		db.mu.Unlock()
		return nil
	}
	db.closed = true
	db.mu.Unlock()

	db.mergeWG.Wait()
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.closeResources()
}

func (db *DB) closeResources() error {
	var closeErr error
	if db.activeFile != nil {
		closeErr = errors.Join(closeErr, db.activeFile.Sync())
	}

	if db.activeFile != nil && db.options.IndexerType == BPlusTree {
		seqNoFile, err := data.OpenSeqNoFile(db.options.DirPath)
		if err != nil {
			closeErr = errors.Join(closeErr, err)
		} else {
			record := &data.LogRecord{
				Key:   []byte(seqNoKey),
				Value: []byte(strconv.FormatUint(db.seqNo, 10)),
			}
			encRecord, _ := data.EncodeLogRecord(record)
			closeErr = errors.Join(closeErr, seqNoFile.Write(encRecord))
			closeErr = errors.Join(closeErr, seqNoFile.Sync())
			closeErr = errors.Join(closeErr, seqNoFile.Close())
		}
	}

	if db.index != nil {
		closeErr = errors.Join(closeErr, db.index.Close())
	}
	if db.activeFile != nil {
		closeErr = errors.Join(closeErr, db.activeFile.Close())
		db.activeFile = nil
	}
	for _, file := range db.olderFiles {
		closeErr = errors.Join(closeErr, file.Close())
	}
	db.olderFiles = make(map[uint32]*data.DataFile)
	if db.fileLock != nil && db.fileLock.Locked() {
		closeErr = errors.Join(closeErr, db.fileLock.Unlock())
	}
	return closeErr
}

func (db *DB) cleanupOpenFailure() error {
	var cleanupErr error
	if db.index != nil {
		cleanupErr = errors.Join(cleanupErr, db.index.Close())
	}
	if db.activeFile != nil {
		cleanupErr = errors.Join(cleanupErr, db.activeFile.Close())
		db.activeFile = nil
	}
	for _, file := range db.olderFiles {
		cleanupErr = errors.Join(cleanupErr, file.Close())
	}
	db.olderFiles = make(map[uint32]*data.DataFile)
	if db.fileLock != nil && db.fileLock.Locked() {
		cleanupErr = errors.Join(cleanupErr, db.fileLock.Unlock())
	}
	return cleanupErr
}

// Sync 持久化数据文件
func (db *DB) Sync() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.closed {
		return ErrClosed
	}
	if db.activeFile == nil {
		return nil
	}
	return db.activeFile.Sync()
}

// 返回数据库的相关统计信息
func (db *DB) Stat() (*Stat, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	if db.closed {
		return nil, ErrClosed
	}
	var dataFiles = uint(len(db.olderFiles))
	if db.activeFile != nil {
		dataFiles++
	}
	dirSize, err := utils.DirSize(db.options.DirPath)
	if err != nil {
		return nil, err
	}
	keyNum, err := db.index.Size()
	if err != nil {
		return nil, err
	}
	return &Stat{
		KeyNum:          uint(keyNum),
		DataFileNum:     dataFiles,
		ReclaimableSize: db.reclaimSize,
		DiskSize:        dirSize,
	}, nil
}

func (db *DB) Backup(dir string) error {
	db.mu.RLock()
	defer db.mu.RUnlock()
	if db.closed {
		return ErrClosed
	}
	return utils.CopyDir(db.options.DirPath, dir, []string{fileLockName})
}

// Put 写入key/value数据，key不能为空
func (db *DB) Put(key []byte, value []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.closed {
		return ErrClosed
	}
	// 判断key是否有效
	if len(key) == 0 {
		return ErrKeyIsEmpty
	}

	// 构造 LogRecord 结构体
	logRecord := &data.LogRecord{
		Key:   logRecordKeyWithSeq(key, nonTransactionSeqNo),
		Value: value,
		Type:  data.LogRecordNormal,
	}

	// 追加写入到当前活跃数据文件当中
	pos, err := db.appendLogRecord(logRecord)
	if err != nil {
		return err
	}
	// fmt.Println("key  = ", key, "pos = ", pos)
	// 更新内存索引
	oldPos, err := db.index.Put(key, pos)
	if err != nil {
		return err
	}
	if oldPos != nil {
		db.reclaimSize += int64(oldPos.Size)
		return nil
	}
	return nil

}

func (db *DB) Get(key []byte) ([]byte, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	if db.closed {
		return nil, ErrClosed
	}
	// 判断key是否有效
	if len(key) == 0 {
		return nil, ErrKeyIsEmpty
	}

	// 从内存数据结构中取出 key 对应的索引信息
	logRecordPos, err := db.index.Get(key)
	if err != nil {
		return nil, err
	}
	if logRecordPos == nil {
		return nil, ErrKeyNotFound
	}

	// 从数据文件中获取 value
	return db.getValueByPosition(logRecordPos)

}

// 获取数据库中所有的 key
func (db *DB) ListKeys() (keys [][]byte, retErr error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	if db.closed {
		return nil, ErrClosed
	}
	iterator, err := db.index.Iterator(false)
	if err != nil {
		return nil, err
	}
	defer func() { retErr = errors.Join(retErr, iterator.Close()) }()
	size, err := db.index.Size()
	if err != nil {
		return nil, err
	}
	keys = make([][]byte, size)
	var idx int
	for iterator.Rewind(); iterator.Valid(); iterator.Next() {
		keys[idx] = iterator.Key()
		idx++
	}
	return keys, nil
}

// 获取所有的数据，并执行用户指定的操作, 函数返回 false 则停止迭代
func (db *DB) Fold(fn func(key []byte, value []byte) bool) (retErr error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	if db.closed {
		return ErrClosed
	}
	iterator, err := db.index.Iterator(false)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, iterator.Close()) }()
	for iterator.Rewind(); iterator.Valid(); iterator.Next() {
		key := iterator.Key()
		value, err := db.getValueByPosition(iterator.Value())
		if err != nil {
			return err
		}
		if !fn(key, value) {
			break
		}
	}
	return nil
}

// 根据索引信息获取对应的 value
func (db *DB) getValueByPosition(logRecordPos *data.LogRecordPos) ([]byte, error) {
	// 根据文件 id 找到对应的数据文件
	var dataFile *data.DataFile

	// 先判断是否为当前活跃文件，如果不是就在就文件里找
	if db.activeFile != nil && db.activeFile.FileId == logRecordPos.Fid {
		dataFile = db.activeFile
	} else {
		dataFile = db.olderFiles[logRecordPos.Fid]
	}
	if dataFile == nil {
		return nil, ErrDataFileNotFound
	}

	// 根据偏移量读取数据
	logRecord, _, err := dataFile.ReadLogRecord(logRecordPos.Offset)
	if err != nil {
		return nil, err
	}
	// 如果读取到的数据是删除的，则返回key不存在的错误
	if logRecord.Type == data.LogRecordDeleted {
		return nil, ErrKeyNotFound
	}
	return logRecord.Value, nil
}

func (db *DB) Delete(key []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.closed {
		return ErrClosed
	}
	// 判断key是否有效
	if len(key) == 0 {
		return ErrKeyIsEmpty
	}
	// 检查key是否存在
	indexPos, err := db.index.Get(key)
	if err != nil {
		return err
	}
	if indexPos == nil {
		return nil
	}
	// 构造删除的 LogRecord
	logRecord := &data.LogRecord{
		Key:   logRecordKeyWithSeq(key, nonTransactionSeqNo),
		Value: nil,
		Type:  data.LogRecordDeleted,
	}
	// 追加写入到当前活跃文件
	pos, err := db.appendLogRecord(logRecord)
	if err != nil {
		return err
	}
	// delete 这条数据本身也可以进行删除
	db.reclaimSize += int64(pos.Size)
	// 从内存索引中删除
	oldPos, ok, err := db.index.Delete(key)
	if err != nil {
		return err
	}
	if !ok {
		return ErrIndexUpdateFailed
	}
	db.reclaimSize += int64(oldPos.Size)
	return nil
}

// 追加写数据到活跃文件中
func (db *DB) appendLogRecord(logRecord *data.LogRecord) (*data.LogRecordPos, error) {

	// 判断当前活跃文件是否存在 因为数据库在没有写入时是没有文件生成的
	// 如果为空则初始化数据文件
	if db.activeFile == nil {
		if err := db.setActiveDataFile(); err != nil {
			return nil, err
		}
	}
	// 写入数据编码
	encRecord, size := data.EncodeLogRecord(logRecord)

	// 如果写入的数据已经达到了活跃文件的阈值，则关闭活跃文件，并创建新的活跃文件
	if db.activeFile.WriteOff+size > db.options.DataFileSize {
		// 先持久化此数据文件，保证已有的文件持久化到磁盘中
		if err := db.activeFile.Sync(); err != nil {
			return nil, err
		}

		// 将当前活跃文件转化为一个旧的文件
		db.olderFiles[db.activeFile.FileId] = db.activeFile

		// 打开新的活跃文件
		if err := db.setActiveDataFile(); err != nil {
			return nil, err
		}
	}

	writeOff := db.activeFile.WriteOff
	if err := db.activeFile.Write(encRecord); err != nil {
		return nil, err
	}
	db.bytesWrite += uint(size)

	// 判断是否需要直接进行持久化
	var needSync = db.options.SyncWrites
	if !needSync && db.options.BytesPerSync > 0 && db.bytesWrite >= db.options.BytesPerSync {
		needSync = true
	}
	if needSync {
		if err := db.activeFile.Sync(); err != nil {
			return nil, err
		}
		if db.bytesWrite > 0 {
			db.bytesWrite = 0
		}
	}

	// 构造内存索引信息
	pos := &data.LogRecordPos{Fid: db.activeFile.FileId, Offset: writeOff, Size: uint32(size)}
	return pos, nil

}

// 设置当前活跃文件
// 在访问此方法前必须有互斥锁
func (db *DB) setActiveDataFile() error {
	var initialFileId uint32 = 0
	// 新的活跃文件id都是递增的
	if db.activeFile != nil {
		initialFileId = db.activeFile.FileId + 1
	}
	// 打开新的数据文件

	dataFile, err := data.OpenDataFile(db.options.DirPath, initialFileId, fio.StandardFIO)
	if err != nil {
		return err
	}
	db.activeFile = dataFile
	return nil
}

func checkOptions(options *Options) error {
	if options == nil {
		return fmt.Errorf("%w: options is nil", ErrInvalidOptions)
	}
	if options.DirPath == "" {
		return fmt.Errorf("%w: database dir path is empty", ErrInvalidOptions)
	}
	if options.DataFileSize <= 0 {
		return fmt.Errorf("%w: database data file size must be greater than 0", ErrInvalidOptions)
	}
	if math.IsNaN(float64(options.DataFileMergeRatio)) || options.DataFileMergeRatio < 0 || options.DataFileMergeRatio > 1 {
		return fmt.Errorf("%w: database data file merge ratio must be between 0 and 1", ErrInvalidOptions)
	}
	if options.IndexerType < BTree || options.IndexerType > BPlusTree {
		return fmt.Errorf("%w: index type %d", ErrUnsupported, options.IndexerType)
	}
	return nil
}

// 从磁盘加载数据文件
func (db *DB) loadDataFiles() error {
	dirEntries, err := os.ReadDir(db.options.DirPath)
	if err != nil {
		return err
	}
	var fileIds []int

	// 遍历目录中的所有文件， 找到所有以.data结尾的文件
	for _, entry := range dirEntries {
		if strings.HasSuffix(entry.Name(), data.DataFileNameSuffix) {
			splitNames := strings.Split(entry.Name(), ".") // xx.data
			fileId, err := strconv.Atoi(splitNames[0])
			if err != nil {
				return fmt.Errorf("%w: invalid data file name %q", ErrCorrupted, entry.Name())
			}
			fileIds = append(fileIds, fileId)
		}
	}
	// 对文件进行排序，从小到大依次加载
	sort.Ints(fileIds)
	db.fileIds = fileIds
	// 遍历每个文件id，打开对应的数据文件
	for i, fileId := range fileIds {

		ioType := fio.StandardFIO
		if db.options.MMapAtStartup {
			ioType = fio.MemoryMap
		}

		dataFile, err := data.OpenDataFile(db.options.DirPath, uint32(fileId), byte(ioType))
		if err != nil {
			return err
		}
		if i == len(fileIds)-1 { // 最后一个是活跃文件
			db.activeFile = dataFile
		} else {
			db.olderFiles[uint32(fileId)] = dataFile
		}
	}
	return nil
}

// 从数据文件中加载索引
// 遍历文件的所有记录，更新到内存索引中
func (db *DB) loadIndexFromDataFiles() error {
	if len(db.fileIds) == 0 {
		return nil
	}

	// 查看是否发生过 merge
	hasMerge, nonMergeFileId := false, uint32(0)
	mergeFinFileName := filepath.Join(db.options.DirPath, data.MergeFinishedFileName)
	if _, err := os.Stat(mergeFinFileName); err == nil {
		fid, err := db.getNonMergeFileId(db.options.DirPath)
		if err != nil {
			return err
		}
		hasMerge = true
		nonMergeFileId = fid
	}

	updateIndex := func(key []byte, typ data.LogRecordType, pos *data.LogRecordPos) error {
		var oldPos *data.LogRecordPos
		var err error
		if typ == data.LogRecordDeleted {
			oldPos, _, err = db.index.Delete(key)
			db.reclaimSize += int64(pos.Size)
		} else {
			oldPos, err = db.index.Put(key, pos)
		}
		if err != nil {
			return err
		}
		if oldPos != nil {
			db.reclaimSize += int64(oldPos.Size)
		}
		return nil
	}

	// 暂存事务数据
	transactionRecords := make(map[uint64][]*data.TransactionRecord)
	var currentSeqNo = nonTransactionSeqNo

	// 遍历所有文件id，处理文件中的记录
	for i, fid := range db.fileIds {
		var fileId = uint32(fid)
		// 如果比最近未参与 merge 的文件 id 更小， 说明已经从 hint 文件中加载
		if hasMerge && fileId < nonMergeFileId {
			continue
		}
		var dataFile *data.DataFile
		if fileId == db.activeFile.FileId {
			dataFile = db.activeFile
		} else {
			dataFile = db.olderFiles[fileId]
		}

		var offset int64 = 0
		for {
			logRecord, size, err := dataFile.ReadLogRecord(offset)
			if err != nil {
				if err == io.EOF {
					break
				}
				return err
			}
			// 构造内存索引并保存
			logRecordPos := &data.LogRecordPos{Fid: fileId, Offset: offset, Size: uint32(size)}

			// 解析 key,拿到事务序列号
			realKey, seqNo := parseLogRecordKey(logRecord.Key)

			// 非事务操作，直接更新内存索引
			if seqNo == nonTransactionSeqNo {
				if err := updateIndex(realKey, logRecord.Type, logRecordPos); err != nil {
					return err
				}
			} else { // 事务操作，只有当获取到事务完成标识时，对应的 seqNo 的数据才可以更新到内存索引中
				if logRecord.Type == data.LogRecordTxnFinished {
					for _, txnRecord := range transactionRecords[seqNo] {
						if err := updateIndex(txnRecord.Record.Key, txnRecord.Record.Type, txnRecord.Pos); err != nil {
							return err
						}
					}
					delete(transactionRecords, seqNo)
				} else {
					logRecord.Key = realKey
					transactionRecords[seqNo] = append(transactionRecords[seqNo], &data.TransactionRecord{
						Pos:    logRecordPos,
						Record: logRecord,
					})
				}
			}

			// 更新事务序列号
			if seqNo > currentSeqNo {
				currentSeqNo = seqNo
			}

			// 递增offset， 下一次从新的位置开始读取
			offset += size
		}
		// 如果是当前活跃文件，更新 WriteOff
		if i == len(db.fileIds)-1 {
			db.activeFile.WriteOff = offset
		}
	}

	// 更新事务序列号  --> 下次事务操作会从当前位置开始进行
	db.seqNo = currentSeqNo

	return nil
}

func (db *DB) loadSeqNo() error {
	fileName := filepath.Join(db.options.DirPath, data.SeqNoFileName)
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		return nil
	}
	seqNoFile, err := data.OpenSeqNoFile(db.options.DirPath)
	if err != nil {
		return err
	}

	record, _, err := seqNoFile.ReadLogRecord(0)
	if err != nil {
		return errors.Join(err, seqNoFile.Close())
	}
	seqNo, err := strconv.ParseUint(string(record.Value), 10, 64)
	if err != nil {
		return errors.Join(err, seqNoFile.Close())
	}
	if err := seqNoFile.Close(); err != nil {
		return err
	}
	db.seqNo = seqNo
	db.seqNoFileExists = true

	return os.Remove(fileName)
}

// 将数据文件 IO 类型设置为标准文件 IO
func (db *DB) resetIoType() error {
	if db.activeFile == nil {
		return nil
	}
	if err := db.activeFile.SetIOManager(db.options.DirPath, fio.StandardFIO); err != nil {
		return err
	}
	for _, dataFile := range db.olderFiles {
		if err := dataFile.SetIOManager(db.options.DirPath, fio.StandardFIO); err != nil {
			return err
		}
	}
	return nil
}
