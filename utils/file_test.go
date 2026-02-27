package utils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDirSize(t *testing.T) {
	dir, _ := os.Getwd()
	size, _ := DirSize(dir)
	assert.True(t, size > 0)
}

func TestAvailableDiskSize(t *testing.T) {
	size, err := AvailableDIskSize()
	assert.Nil(t, err)
	t.Log(size / 1024 / 1024 / 1024)
	assert.True(t, size > 0)
}
