package bitcask_go

import (
	"bitcask-go/data"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func checkCurrentDataSIze(dataDir string) (int64, error) {
	var totalSize int64
	err := filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, data.DataFileNameSuffix) {
			totalSize += info.Size()
			fmt.Printf("File %s size: %d MB\n",
				filepath.Base(path), info.Size()/1024/1024)
		}
		return nil
	})
	return totalSize, err
}

func GenerateBitcaskTestData(db *DB, targetSizeGB int) error {
	targetSize := int64(targetSizeGB) * 1024 * 1024 * 1024
	var totalSize int64

	// 模拟真实场景的key-value分布
	keyPatterns := []string{
		"user:%d",
		"session:%d",
		"cache:%d",
		"metric:%d:%d",
	}

	// 不同大小的value模拟真实场景
	valueSizes := []int{100, 500, 1024, 2048, 5120} // 100B到5KB

	start := time.Now()
	for i := 0; totalSize < targetSize; i++ {
		// 随机选择key模式和value大小
		pattern := keyPatterns[i%len(keyPatterns)]
		valueSize := valueSizes[i%len(valueSizes)]

		var key string
		if strings.Contains(pattern, "%d:%d") {
			key = fmt.Sprintf(pattern, i/1000, i%1000)
		} else {
			key = fmt.Sprintf(pattern, i)
		}

		// 生成指定大小的value
		value := make([]byte, valueSize)
		for j := range value {
			value[j] = byte((i + j) % 256)
		}

		err := db.Put([]byte(key), value)
		if err != nil {
			return err
		}

		// Bitcask会自动计算实际存储大小（包括header等）
		totalSize += int64(len(key) + valueSize + 16) // 大概的开销估算

		if i%10000 == 0 {
			elapsed := time.Since(start)
			progress := float64(totalSize) / float64(targetSize) * 100
			fmt.Printf("Progress: %.2f%% (%d MB), Keys: %d, Elapsed: %v\n",
				progress, totalSize/1024/1024, i, elapsed)
		}
	}
	return nil
}

func TestMMapPerformance() error {
	dir := "/tmp/bitcask-go-mmap"
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		fmt.Printf("创建临时目录失败： %v\n", err)
		return err
	}
	fmt.Printf("临时目录创建在： %s\n", dir)
	currentSize, err := checkCurrentDataSIze(dir)
	if err != nil {
		return err
	}
	fmt.Printf("Current data size: %d MB\n", currentSize/1024/1024)
	if currentSize < 1024*1024*1024 {
		opts := DefaultOptions()

		opts.DirPath = dir
		db, err := Open(&opts)
		if err != nil {
			return err
		}
		if err := GenerateBitcaskTestData(db, 4); err != nil {
			_ = db.Close()
			return err
		}
		if err := db.Close(); err != nil {
			return err
		}
	}
	fmt.Println("Testing index loading with MMap...")
	start := time.Now()
	opts := DefaultOptions()
	opts.DirPath = dir
	opts.MMapAtStartup = true
	db, err := Open(&opts)
	if err != nil {
		return err
	}
	loadTime := time.Since(start)
	fmt.Printf("Index loading took: %v\n", loadTime)
	if err := db.Close(); err != nil {
		return err
	}

	fmt.Println("Testing index loading with StanderIO...")
	start = time.Now()
	opts = DefaultOptions()
	opts.DirPath = dir
	opts.MMapAtStartup = false
	db, err = Open(&opts)
	if err != nil {
		return err
	}
	loadTime = time.Since(start)
	fmt.Printf("Index loading took: %v\n", loadTime)
	if err := db.Close(); err != nil {
		return err
	}
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	return nil
}
