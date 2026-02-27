package bitcask_go

import (
	"bitcask-go/data"
	"bitcask-go/utils"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
)

const (
	mergeDirName     = "-merge"
	mergeFinishedKey = "merge.finished"
)

func (db *DB) Merge() error {
	// 如果活跃数据文件为空，直接返回(数据库为空)
	if db.activeFile == nil {
		return nil
	}
	db.mu.Lock()

	// 如果 Merge 正在进行中，直接返回
	if db.isMerging {
		db.mu.Unlock()
		return ErrMergIsProcess
	}

	// 查看失效数据量是否达到需要merge的阈值
	totalSize, err := utils.DirSize(db.options.DirPath)
	if err != nil {
		db.mu.Unlock()
		return err
	}
	if float32(db.reclaimSize) / float32(totalSize) < db.options.DataFileMergeRatio {
		db.mu.Unlock()
		return ErrMergeRatioUnreached
	}

	availableSize, err := utils.AvailableDIskSize()
	if err != nil {
		db.mu.Unlock()
		return err
	}


	// 磁盘剩余空间是否可以继续执行Merge
	if uint64(totalSize - db.reclaimSize) >= availableSize {
		db.mu.Unlock()
		return ErrDataDirectoryCorrupted
	}


	db.isMerging = true
	defer func() {
		db.isMerging = false
	}()

	// 持久化当前活跃文件   ---> 如果在执行 Merge 时，还在进行Put(也在使用当前活跃文件 是否存在问题？)
	// 不会， 因为在执行Merge拿取所有数据前，有对db进行加锁(和Put竞争db的使用权)
	if err := db.activeFile.Sync(); err != nil {
		db.mu.Unlock()
		return err
	}

	// 将当前活跃文件转成旧的数据文件
	db.olderFiles[db.activeFile.FileId] = db.activeFile

	// 打开新的活跃文件
	if err := db.setActiveDataFile(); err != nil {
		db.mu.Unlock()
		return err
	}

	// 记录最近没有参与 merge 的文件id
	nonMergeFileId := db.activeFile.FileId
	// TODO
	// 是否会 存在这么一个问题， 如果后续代码报错，即没有完成Merge，
	// 由于nonMergeFileId，我会认为小于该值的文件都是无效文件

	// 所有需要merge的文件
	var mergeFiles []*data.DataFile
	for _, file := range db.olderFiles {
		mergeFiles = append(mergeFiles, file)
	}
	// 保存完所有待Merge文件后就可以释放锁，减少锁等待时间
	db.mu.Unlock()
	

	// 待 merge 的文件按照ID从小到大进行排序，依次 merge
	sort.Slice(mergeFiles, func(i, j int) bool {
		return mergeFiles[i].FileId < mergeFiles[j].FileId

	})

	mergePath := db.getMergePath()
	// 如果之前存在就存在该目录，说明之前发生过 merge， 将其删除掉
	if _, err := os.Stat(mergePath); err == nil {
		if err := os.RemoveAll(mergePath); err != nil {
			return err
		}
	}
	// 新建一个 mergePath的目录
	if err := os.MkdirAll(mergePath, os.ModePerm); err != nil {
		return err
	}

	// 打开一个新的临时 bitcask 实例
	mergeOptions := db.options
	mergeOptions.DirPath = mergePath
	mergeOptions.SyncWrites = false // 只在Merge结束时再调用依次Sync操作
	mergeDB, err := Open(mergeOptions)
	if err != nil {
		return err
	}
	// 打开一个 hint 文件存储索引
	hintFile, err := data.OpenHintFile(mergePath)
	if err != nil {
		return err
	}
	// 遍历所有的数据文件，将数据写入到 mergeDB 中
	for _, dataFile := range mergeFiles {
		var offset int64 = 0
		for {
			logRecord, size, err := dataFile.ReadLogRecord(offset)
			if err != nil {
				if err == io.EOF {
					break
				}
				return err

			}
			realKey, _ := parseLogRecordKey(logRecord.Key)
			logRecordPos := db.index.Get(realKey)
			// 和内存中的索引位置进行比较，如果有效则重写
			if logRecordPos != nil &&
				logRecordPos.Fid == dataFile.FileId &&
				logRecordPos.Offset == offset {
				// 清除事务标记
				logRecord.Key = logRecordKeyWithSeq(realKey, nonTransactionSeqNo)

				// 重写 这里重写为什么不用加Lock？
				// 因为这里是新开了一个db实例，不存在锁竞争以及并发问题
				pos, err := mergeDB.appendLogRecord(logRecord)
				if err != nil {
					return err
				}

				// 将当前位置索引写到 Hint文件当中
				if err := hintFile.WriteHintRecord(realKey, pos); err != nil {
					return err
				}
			}
			offset += size
		}
	}


	if err := hintFile.Sync(); err != nil {
		return err
	}
	if err := mergeDB.Sync(); err != nil {
		return err
	}

	// 写标识 merge 完成的文件
	mergeFinishedFile, err := data.OpenMergeFinishedFile(mergePath)
	if err != nil {
		return err
	}
	mergeFinRecord := &data.LogRecord{
		Key:   []byte(mergeFinishedKey),
		Value: []byte(strconv.Itoa(int(nonMergeFileId))),
	}
	encRecord, _ := data.EncodeLogRecord(mergeFinRecord)
	if err := mergeFinishedFile.Write(encRecord); err != nil {
		return err
	}

	if err := mergeFinishedFile.Sync(); err != nil {
		return err
	}

	return nil
}

// /tmp/bitcask

func (db *DB) getMergePath() string {
	dir := path.Dir(path.Clean(db.options.DirPath))
	base := path.Base(db.options.DirPath)
	return filepath.Join(dir, base+mergeDirName)
}

// 加载 mergr 数据目录
func (db *DB) LoadMergeFiles() error {
	mergePath := db.getMergePath()
	// 不存在则直接返回
	if _, err := os.Stat(mergePath); os.IsNotExist(err) {
		return nil
	}
	defer func() {
		_ = os.RemoveAll(mergePath)
	}()
	dirEntries, err := os.ReadDir(mergePath)
	if err != nil {
		return err
	}
	// 查询标识 merge 完成的文件， 判断 merge 是否处理完成
	var mergeFinished bool
	var mergeFileNames []string

	for _, entry := range dirEntries {
		if entry.Name() == data.MergeFinishedFileName {
			mergeFinished = true
		}
		// 这个操作是干什么的？ 好像是错误的
		if entry.Name() == data.SeqNoFileName {
			continue
		}
		if entry.Name() == fileLockName {
			continue
		}
		mergeFileNames = append(mergeFileNames, entry.Name())
	}
	// 没有 merge 完成则直接返回
	if !mergeFinished {
		return nil
	}
	// 获取最近没有参与Merge的文件ID
	nonMergeFileId, err := db.getNonMergeFileId(mergePath)
	if err != nil {
		return err
	}

	// 删除旧的数据文件
	var fileId uint32 = 0
	for ; fileId < nonMergeFileId; fileId++ {
		fileName := data.GetDataFileName(db.options.DirPath, fileId)
		if _, err := os.Stat(fileName); err == nil {
			if err := os.Remove(fileName); err != nil {
				return err
			}
		}
	}
	// 将新的数据文件移动到数据目录中
	for _, fileName := range mergeFileNames {
		srcPath := filepath.Join(mergePath, fileName)
		dataPath := filepath.Join(db.options.DirPath, fileName)
		if err := os.Rename(srcPath, dataPath); err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) getNonMergeFileId(dirPath string) (uint32, error) {
	mergeFishedFile, err := data.OpenMergeFinishedFile(dirPath)
	if err != nil {
		return 0, err
	}
	record, _, err := mergeFishedFile.ReadLogRecord(0)
	if err != nil {
		return 0, err
	}
	nonMergeFileId, err := strconv.Atoi(string(record.Value))
	if err != nil {
		return 0, err
	}
	return uint32(nonMergeFileId), nil
}

func (db *DB) loadIndexFromHintFile() error {
	// hint文件只有一个 我在Merge的时候是否会存在一个hint文件存储不够的情况发生呢？
	hintFileName := filepath.Join(db.options.DirPath, data.HintFileName)
	if _, err := os.Stat(hintFileName); os.IsNotExist(err) {
		return nil
	}
	// 打开 hint索引文件
	hintFile, err := data.OpenHintFile(db.options.DirPath)
	if err != nil {
		return err
	}
	// 读取文件中的索引
	var offset int64 = 0
	for {
		logRecord, size, err := hintFile.ReadLogRecord(offset)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		// 解码拿到实际的位置索引信息
		pos := data.DecodeLogRecordPos(logRecord.Value)
		db.index.Put(logRecord.Key, pos)
		offset += size
	}
	return nil
}
