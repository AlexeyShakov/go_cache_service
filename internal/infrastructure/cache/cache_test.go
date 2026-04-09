package cache

import "testing"

// TestGetByKeyNoKey проверка, что в кэше нет ключа
func TestGetByKeyNoKey(t *testing.T) {
	cache := NewInMemoryCache()

	key := "key1"
	_, ok := cache.GetByKey(key)
	if ok {
		t.Fatalf("There must be no value with key=%q", key)
	}
}

// TestAddKey проверка, что после добавления ключ есть в кэше
func TestAddKey(t *testing.T) {
	cache := NewInMemoryCache()
	testKey, testValue := "key", "value"
	cache.AddKey(testKey, testValue)

	val, ok := cache.GetByKey(testKey)
	if !ok {
		t.Fatalf("There is no key=%q in the cache", testKey)
	}
	if val != testValue {
		t.Fatalf("The value=%q of the key=%v is not equal to expected value=%s", val, testKey, testValue)
	}

}

// TestGetAllKeys проверка, что получаем все ключи из кэша
func TestGetAllKeys(t *testing.T) {
	cache := NewInMemoryCache()
	testKey1, testValue1 := "key1", "value1"
	testKey2, testValue2 := "key12", "value2"

	keysMap := map[string]struct{}{testKey1: {}, testKey2: {}}

	cache.AddKey(testKey1, testValue1)
	cache.AddKey(testKey2, testValue2)

	allKeys := cache.GetAllKeys()
	if len(allKeys) != len(keysMap) {
		t.Fatalf("Different amount of keys!")
	}
	for _, key := range allKeys {
		_, ok := keysMap[key]
		if !ok {
			t.Fatalf("The key=%q must be in the cache", key)
		}
	}

}

// TestReplaceKeys проверка, что заменяем значения у переданных ключей. Значения у ключей для, которых мы ничего не передали
// не должны поменяться
func TestReplaceKeys(t *testing.T) {
	notReplacedKey, notReplacedValue := "key1", "value1"
	ReplacedKey1, val1 := "key2", "value2"
	ReplacedKey2, val2 := "key3", "value3"

	cache := NewInMemoryCache()
	cache.AddKey(notReplacedKey, notReplacedValue)
	cache.AddKey(ReplacedKey1, val1)
	cache.AddKey(ReplacedKey2, val2)

	newValue1 := "newVal1"
	newValue2 := "newVal2"

	cache.ReplaceKeys([]string{ReplacedKey1, ReplacedKey2}, []string{newValue1, newValue2})

	res1, _ := cache.GetByKey(notReplacedKey)
	if res1 != notReplacedValue {
		t.Fatalf("The value for this key=%s should not have been changed", notReplacedKey)
	}
	res2, _ := cache.GetByKey(ReplacedKey1)
	if res2 != newValue1 {
		t.Fatalf("The value=%s for this key=%s is not expected=%s", res2, ReplacedKey1, newValue1)
	}
	res3, _ := cache.GetByKey(ReplacedKey2)
	if res3 != newValue2 {
		t.Fatalf("The value=%s for this key=%s is not expected=%s", res3, ReplacedKey2, newValue2)
	}
}
