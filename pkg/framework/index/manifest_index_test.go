package index

import (
	"testing"
)

func TestIndexGet(t *testing.T) {
	idx := NewIndex()
	testKey := "default/Deployment/test"
	testValue := []byte("test value")

	idx.Set(testKey, testValue)

	val, ok := idx.Get(testKey)
	if !ok {
		t.Fatal("expected key to exist")
	}
	if string(val) != string(testValue) {
		t.Errorf("expected %s, got %s", string(testValue), string(val))
	}
}

func TestIndexGetNotFound(t *testing.T) {
	idx := NewIndex()
	testKey := "default/Deployment/test"

	_, ok := idx.Get(testKey)
	if ok {
		t.Fatal("expected key to not exist")
	}
}

func TestIndexSet(t *testing.T) {
	idx := NewIndex()
	testKey := "default/Deployment/test"
	testValue := []byte("test value")

	idx.Set(testKey, testValue)

	val, ok := idx.Get(testKey)
	if !ok {
		t.Fatal("expected key to exist")
	}
	if string(val) != string(testValue) {
		t.Errorf("expected %s, got %s", string(testValue), string(val))
	}
}

func TestIndexDelete(t *testing.T) {
	idx := NewIndex()
	testKey := "default/Deployment/test"
	testValue := []byte("test value")

	idx.Set(testKey, testValue)
	idx.Delete(testKey)

	_, ok := idx.Get(testKey)
	if ok {
		t.Fatal("expected key to be deleted")
	}
}

func TestIndexList(t *testing.T) {
	idx := NewIndex()
	testKey1 := "default/Deployment/test1"
	testValue1 := []byte("test value 1")
	testKey2 := "default/Service/test2"
	testValue2 := []byte("test value 2")

	idx.Set(testKey1, testValue1)
	idx.Set(testKey2, testValue2)

	list := idx.List()
	if len(list) != 2 {
		t.Errorf("expected 2 items, got %d", len(list))
	}

	if string(list[testKey1]) != string(testValue1) {
		t.Errorf("expected %s, got %s", string(testValue1), string(list[testKey1]))
	}
	if string(list[testKey2]) != string(testValue2) {
		t.Errorf("expected %s, got %s", string(testValue2), string(list[testKey2]))
	}
}

func TestIndexMerge(t *testing.T) {
	idx := NewIndex()
	embedded := map[string][]byte{
		"default/Deployment/app1": []byte("app1"),
	}
	overrides := map[string][]byte{
		"default/Deployment/app1": []byte("app1-override"),
		"default/Deployment/app2": []byte("app2"),
	}

	idx.Merge(embedded, overrides)

	val, _ := idx.Get("default/Deployment/app1")
	if string(val) != "app1-override" {
		t.Errorf("expected override to take precedence, got %s", string(val))
	}

	val, _ = idx.Get("default/Deployment/app2")
	if string(val) != "app2" {
		t.Errorf("expected app2 to exist, got %s", string(val))
	}
}

func TestIndexConcurrentAccess(t *testing.T) {
	idx := NewIndex()
	done := make(chan bool)

	go func() {
		for i := 0; i < 100; i++ {
			key := "default/Deployment/test" + string(rune(i))
			idx.Set(key, []byte("value"))
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			key := "default/Service/test" + string(rune(i))
			idx.Set(key, []byte("value"))
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			idx.List()
		}
		done <- true
	}()

	<-done
	<-done
	<-done
}

