package bitcask_go

import (
	"testing"
)

// 测试使用MMap内存映射替换标准文件IO对db实例加载的提升速度
func TestMMapPerformanceTest(t *testing.T) {
	t.Skip("manual multi-gigabyte startup experiment; use the benchmark package for routine measurements")
}
