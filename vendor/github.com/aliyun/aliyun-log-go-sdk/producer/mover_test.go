package producer

import (
	"fmt"
	"sync"
	"testing"
)

func TestMoverRange(t *testing.T) {
	testMap := &sync.Map{}
	for i := 0; i < 10; i++ {
		testMap.Store(fmt.Sprintf("key %v", i), "value")
	}
	testMap.Range(func(key, batch interface{}) bool {
		testMap.Delete(key)
		return true
	})
	if syncMapCount(testMap) != 0 {
		t.Error("Can't be deleted in syncMap")
	}
}

func syncMapCount(syncMap *sync.Map) int {
	count := 0
	syncMap.Range(func(key, batch interface{}) bool {
		count += 1
		return true
	})
	return count
}
