package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTestKey(t *testing.T) {
	for i := 0; i < 10; i++ {
		assert.NotNil(t, string(GetTestKey(i)))
	}
}


func TestGetTestValue(t *testing.T) {
	for i := 0; i < 10; i++ {
		assert.NotNil(t, string(GetTestKey(i)))
		t.Log(string(RandomValue(10)))
		assert.NotNil(t, string(RandomValue(10)))
	}
}