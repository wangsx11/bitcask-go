package benchmark

import (
	bitcask "bitcask-go"
	"bitcask-go/utils"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var db *bitcask.DB

func init() {
	// 初始化存储引擎实例
	options := bitcask.DefaultOptions
	dir, _ := os.MkdirTemp("", "bitcask-go-benchmark")
	options.DirPath = dir

	var err error
	db, err = bitcask.Open(options)
	if err != nil {
		panic(fmt.Sprintf("failed to open db: %v", err))
	}

}
 
func Benchmark_Put(b *testing.B) {

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := db.Put(utils.GetTestKey(i), utils.RandomValue(1024))
		assert.Nil(b, err)
	}
}


func Benchmark_Get(b *testing.B) {
	for i := 0; i < 10000; i++ {
		err := db.Put(utils.GetTestKey(i), utils.RandomValue(1024))
		assert.Nil(b, err)
	}
	rand.Seed(time.Now().UnixNano())
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := db.Get(utils.GetTestKey(rand.Int()))
		if err != nil && err != bitcask.ErrKeyNotFound {
			b.Fatal(err)
		}
	}
}


func Benchmark_Delete(b *testing.B) {
	rand.Seed(time.Now().UnixNano())

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := db.Delete(utils.GetTestKey(rand.Int()))
		assert.Nil(b, err)
	}
}