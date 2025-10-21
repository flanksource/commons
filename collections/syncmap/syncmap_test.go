package syncmap

import (
	"sync"
	"testing"
)

func TestSyncMap_LoadAndStore(t *testing.T) {
	var m SyncMap[string, int]

	m.Store("key1", 100)
	val, ok := m.Load("key1")
	if !ok {
		t.Error("Expected key1 to be present")
	}
	if val != 100 {
		t.Errorf("Expected 100, got %d", val)
	}

	_, ok = m.Load("nonexistent")
	if ok {
		t.Error("Expected nonexistent key to return false")
	}
}

func TestSyncMap_LoadOrStore(t *testing.T) {
	var m SyncMap[string, int]

	actual, loaded := m.LoadOrStore("key1", 100)
	if loaded {
		t.Error("Expected loaded to be false for new key")
	}
	if actual != 100 {
		t.Errorf("Expected 100, got %d", actual)
	}

	actual, loaded = m.LoadOrStore("key1", 200)
	if !loaded {
		t.Error("Expected loaded to be true for existing key")
	}
	if actual != 100 {
		t.Errorf("Expected 100 (original value), got %d", actual)
	}
}

func TestSyncMap_Delete(t *testing.T) {
	var m SyncMap[string, int]

	m.Store("key1", 100)
	m.Delete("key1")

	_, ok := m.Load("key1")
	if ok {
		t.Error("Expected key1 to be deleted")
	}

	m.Delete("nonexistent")
}

func TestSyncMap_LoadAndDelete(t *testing.T) {
	var m SyncMap[string, int]

	m.Store("key1", 100)

	val, loaded := m.LoadAndDelete("key1")
	if !loaded {
		t.Error("Expected loaded to be true")
	}
	if val != 100 {
		t.Errorf("Expected 100, got %d", val)
	}

	_, ok := m.Load("key1")
	if ok {
		t.Error("Expected key1 to be deleted")
	}

	_, loaded = m.LoadAndDelete("nonexistent")
	if loaded {
		t.Error("Expected loaded to be false for non-existent key")
	}
}

func TestSyncMap_CompareAndDelete(t *testing.T) {
	var m SyncMap[string, int]

	m.Store("key1", 100)

	deleted := m.CompareAndDelete("key1", 200)
	if deleted {
		t.Error("Expected delete to fail with wrong value")
	}

	val, ok := m.Load("key1")
	if !ok || val != 100 {
		t.Error("Expected key1 to still exist with value 100")
	}

	deleted = m.CompareAndDelete("key1", 100)
	if !deleted {
		t.Error("Expected delete to succeed with correct value")
	}

	_, ok = m.Load("key1")
	if ok {
		t.Error("Expected key1 to be deleted")
	}

	deleted = m.CompareAndDelete("nonexistent", 100)
	if deleted {
		t.Error("Expected delete to fail for non-existent key")
	}
}

func TestSyncMap_Swap(t *testing.T) {
	var m SyncMap[string, int]

	prev, loaded := m.Swap("key1", 100)
	if loaded {
		t.Error("Expected loaded to be false for non-existent key")
	}
	if prev != 0 {
		t.Errorf("Expected zero value, got %d", prev)
	}

	val, ok := m.Load("key1")
	if !ok || val != 100 {
		t.Error("Expected key1 to have value 100")
	}

	prev, loaded = m.Swap("key1", 200)
	if !loaded {
		t.Error("Expected loaded to be true for existing key")
	}
	if prev != 100 {
		t.Errorf("Expected 100, got %d", prev)
	}

	val, ok = m.Load("key1")
	if !ok || val != 200 {
		t.Error("Expected key1 to have value 200")
	}
}

func TestSyncMap_CompareAndSwap(t *testing.T) {
	var m SyncMap[string, int]

	m.Store("key1", 100)

	swapped := m.CompareAndSwap("key1", 200, 300)
	if swapped {
		t.Error("Expected swap to fail with wrong old value")
	}

	val, ok := m.Load("key1")
	if !ok || val != 100 {
		t.Error("Expected key1 to still have value 100")
	}

	swapped = m.CompareAndSwap("key1", 100, 200)
	if !swapped {
		t.Error("Expected swap to succeed with correct old value")
	}

	val, ok = m.Load("key1")
	if !ok || val != 200 {
		t.Error("Expected key1 to have value 200")
	}

	swapped = m.CompareAndSwap("nonexistent", 100, 200)
	if swapped {
		t.Error("Expected swap to fail for non-existent key")
	}
}

func TestSyncMap_Range(t *testing.T) {
	var m SyncMap[string, int]

	expected := map[string]int{
		"key1": 100,
		"key2": 200,
		"key3": 300,
	}

	for k, v := range expected {
		m.Store(k, v)
	}

	found := make(map[string]int)
	m.Range(func(key string, value int) bool {
		found[key] = value
		return true
	})

	if len(found) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(found))
	}

	for k, v := range expected {
		if found[k] != v {
			t.Errorf("Expected %s=%d, got %d", k, v, found[k])
		}
	}

	count := 0
	m.Range(func(key string, value int) bool {
		count++
		return false // Stop after first iteration
	})

	if count != 1 {
		t.Errorf("Expected Range to stop after 1 iteration, got %d", count)
	}
}

func TestSyncMap_Clear(t *testing.T) {
	var m SyncMap[string, int]

	m.Store("key1", 100)
	m.Store("key2", 200)
	m.Store("key3", 300)

	m.Clear()

	count := 0
	m.Range(func(key string, value int) bool {
		count++
		return true
	})

	if count != 0 {
		t.Errorf("Expected 0 items after Clear, got %d", count)
	}

	_, ok := m.Load("key1")
	if ok {
		t.Error("Expected key1 to be deleted after Clear")
	}
}

func TestSyncMap_Concurrent(t *testing.T) {
	var m SyncMap[int, int]
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			m.Store(val, val*10)
		}(i)
	}

	wg.Wait()

	for i := 0; i < 100; i++ {
		val, ok := m.Load(i)
		if !ok {
			t.Errorf("Expected key %d to exist", i)
		}
		if val != i*10 {
			t.Errorf("Expected %d, got %d", i*10, val)
		}
	}

	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func(val int) {
			defer wg.Done()
			m.Store(val, val*20)
		}(i)
		go func(val int) {
			defer wg.Done()
			m.Load(val)
		}(i)
	}

	wg.Wait()
}
