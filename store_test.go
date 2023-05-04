package bufrex

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	TestDefaultExpiry = 2 * time.Second
)

type TestStorePayload struct {
	age  int
	name string
}

func TestPutWithDefaultExpiry(t *testing.T) {
	store := New(TestDefaultExpiry, 2*time.Minute)
	var name string
	var i int
	name = "key_"
	for i = 0; i < 100; i++ {
		store.Put(name+strconv.Itoa(i), i, DefaultExpiry)
	}

	// value must exists
	value, exist := store.Get("key_90")
	if !exist || value.(int) != 90 {
		t.Errorf("Should exists: Got %d and state existance state is %t", value.(int), exist)
	}

	// value is not expected to exist
	v, e := store.Get("key_99999")
	if e || v != nil {
		t.Errorf("Should not exists: Got %d and state existance is %t", v.(int), e)
	}

	// update value in the map
	store.Put("key_90", 11, DefaultExpiry)

	// get the updated value from cache
	updated_value, _ := store.Get("key_90")
	if updated_value.(int) != 11 {
		t.Errorf("Update from 90 to 11 failed")
	}
}

func TestSet(t *testing.T) {
	// Set will fail if
	store := New(TestDefaultExpiry, 2*time.Minute)
	name1 := "key_1"

	saved1 := store.Set(name1, 1, DefaultExpiry)
	if !saved1 {
		t.Errorf("Failed to save key_1 on first attempt!")
	}

	saved2 := store.Set(name1, 2, DefaultExpiry)
	if saved2 {
		t.Errorf("Update to key_1 should have failed on second attempt")
	}
}

func TestGetExpired(t *testing.T) {
	store := New(-2*time.Second, 2*time.Minute)
	setOnDelete(store)
	name := "key_1"

	store.Put(name, 1, DefaultExpiry)
	v, e := store.Get(name)

	if v != nil || e == true {
		t.Errorf("Cache has already expired")
	}
}

func TestNonExpiringItems(t *testing.T) {
	store := New(-2*time.Second, 2*time.Minute)
	name := "key_1"
	store.Put(name, 1, ExpiryInfinity)
	v, e := store.Get(name)

	if v == nil || e == false {
		t.Errorf("Non expiring item has expired!")
	}
}

func TestDeleteExpired(t *testing.T) {
	store := New(-2*time.Second, 2*time.Minute)
	setOnDelete(store)
	prefix := "key_"
	var i int

	for i = 0; i < 20; i++ {
		store.Put(prefix+strconv.Itoa(i), i, DefaultExpiry)
	}

	if len(store.cache) != 20 {
		t.Errorf("we do not have the correct number of stored values")
	}
	// all has expired already
	store.DeleteExpired()

	if len(store.cache) != 0 {
		t.Errorf("all data in cache should be expired.")
	}
}

func TestCacheCleanup(t *testing.T) {
	store := New(-2*time.Second, 200*time.Millisecond)
	prefix := "key_"
	var i int

	for i = 0; i < 20; i++ {
		store.Put(prefix+strconv.Itoa(i), i, DefaultExpiry)
	}

	if len(store.cache) != 20 {
		t.Errorf("we do not have the correct number of stored values")
	}

	time.Sleep(500 * time.Millisecond)

	if len(store.cache) != 0 {
		t.Errorf("cleanup service failed to clean expired tokens")
	}
}

func TestCacheCleanupAndStop(t *testing.T) {
	store := New(-2*time.Second, 200*time.Millisecond)
	prefix := "key_"
	var i int

	for i = 0; i < 20; i++ {
		store.Put(prefix+strconv.Itoa(i), i, DefaultExpiry)
	}

	if len(store.cache) != 20 {
		t.Errorf("we do not have the correct number of stored values")
	}

	time.Sleep(500 * time.Millisecond)

	if len(store.cache) != 0 {
		t.Errorf("cleanup service failed to clean expired tokens")
	}

	// stop cleanup service
	store.StopCleaner()

	for i = 0; i < 20; i++ {
		store.Put(prefix+strconv.Itoa(i), i, DefaultExpiry)
	}

	if len(store.cache) != 20 {
		t.Errorf("we do not have the correct number of stored values")
	}

	time.Sleep(500 * time.Millisecond)

	if len(store.cache) != 20 {
		t.Errorf("we do not have the correct number of stored values")
	}
}

func TestCachingWithRedis(t *testing.T) {
	encoder := func(_ string, value interface{}) (string, error) {
		return value.(string), nil
	}

	decoder := func(_ string, value string) (interface{}, error) {
		return value, nil
	}

	redis_opts := &redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	}
	store := New(TestDefaultExpiry, 2*time.Minute).ConfigureAdapter("redis", redis_opts, encoder, decoder)
	setOnDelete(store)
	store.Set("expiry_data", "hellew_world", 100*time.Millisecond)
	// check it exists at the redis level as well.
	if _, found := store.redis.GetFromRedis("expiry_data"); !found {
		t.Fatal("Expected to find find `expiry_data` in redis")
	}

	store.Delete("expiry_data")
	if _, found := store.redis.GetFromRedis("expiry_data"); found {
		t.Fatal("`expiry_data` was deleted")
	}
}

func TestConfigureWithWrongRedisParams(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Expected a panic as redis opts is wrong")
		}
	}()

	encoder := func(_ string, value interface{}) (string, error) {
		return value.(string), nil
	}

	decoder := func(_ string, value string) (interface{}, error) {
		return value, nil
	}

	redis_opts := &TestStorePayload{
		name: "some name",
		age:  1232,
	}
	New(TestDefaultExpiry, 2*time.Minute).ConfigureAdapter("redis", redis_opts, encoder, decoder)
}

func TestConfigureWithWrongAdapterName(t *testing.T) {
	encoder := func(_ string, value interface{}) (string, error) {
		return value.(string), nil
	}

	decoder := func(_ string, value string) (interface{}, error) {
		return value, nil
	}

	redis_opts := &redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	}
	store := New(TestDefaultExpiry, 2*time.Minute).ConfigureAdapter("postgres", redis_opts, encoder, decoder)
	if store != nil {
		t.Fatal("wrong adapter type `postgres` was set and should return a nil!")
	}
}

func TestSetupInRedisNotInMemory(t *testing.T) {
	encoder := func(_ string, value interface{}) (string, error) {
		return value.(string), nil
	}

	decoder := func(_ string, value string) (interface{}, error) {
		return value, nil
	}

	redis_opts := &redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	}
	store := New(TestDefaultExpiry, 2*time.Minute).ConfigureAdapter("redis", redis_opts, encoder, decoder)
	// insert datat into redis
	store.redis.SetToRedis("persisted_not_memory", "hello world", 3*time.Second)

	// get data from memory.
	if _, found := store.Get("persisted_not_memory"); !found {
		t.Fatal("Data should be available and obtained from persistent storage")
	}
}

func setOnDelete(store *Store) {
	deleted := func(key string, value interface{}) {
		fmt.Println("just deleted data")
	}
	store.OnDelete(deleted)
}
