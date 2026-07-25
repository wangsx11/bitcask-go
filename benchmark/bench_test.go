package benchmark

import (
	bitcask "bitcask-go"
	"bitcask-go/utils"
	"math/rand"
	"testing"
)

const (
	benchmarkValueSize = 1024
	benchmarkKeyCount  = 10000
)

func openBenchmarkDB(b *testing.B) *bitcask.DB {
	b.Helper()
	opts := *bitcask.DefaultOptions
	opts.DirPath = b.TempDir()

	db, err := bitcask.Open(&opts)
	if err != nil {
		b.Fatalf("open benchmark database: %v", err)
	}
	b.Cleanup(func() {
		if err := db.Close(); err != nil {
			b.Errorf("close benchmark database: %v", err)
		}
	})
	return db
}

func BenchmarkPut(b *testing.B) {
	db := openBenchmarkDB(b)
	value := make([]byte, benchmarkValueSize)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := db.Put(utils.GetTestKey(i), value); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGet(b *testing.B) {
	db := openBenchmarkDB(b)
	value := make([]byte, benchmarkValueSize)
	for i := 0; i < benchmarkKeyCount; i++ {
		if err := db.Put(utils.GetTestKey(i), value); err != nil {
			b.Fatal(err)
		}
	}

	random := rand.New(rand.NewSource(1))
	keys := make([][]byte, benchmarkKeyCount)
	for i := range keys {
		keys[i] = utils.GetTestKey(random.Intn(benchmarkKeyCount))
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := db.Get(keys[i%len(keys)]); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDelete(b *testing.B) {
	db := openBenchmarkDB(b)
	value := make([]byte, benchmarkValueSize)
	for i := 0; i < b.N; i++ {
		if err := db.Put(utils.GetTestKey(i), value); err != nil {
			b.Fatal(err)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := db.Delete(utils.GetTestKey(i)); err != nil {
			b.Fatal(err)
		}
	}
}
