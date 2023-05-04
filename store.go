package bufrex

import (
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// set defaults for no-expiry and default expiry
const (
	DefaultExpiry  = 0
	ExpiryInfinity = -1
)

// A value represents the value of a key/value pair.
// On reaching expiry (which is determined by janitor sweeps or expiry on get),
// The data is deleted from the store
type Value struct {
	expiry int64
	Object interface{}
}

// store struct holds the key value store.
type Store struct {
	defaultExpiration time.Duration
	cache             map[string]Value
	mutex             sync.RWMutex
	onDeleted         func(string, interface{})
	janitor           *cleaner
	redis             *RedisAdapter
}

// func declaration

// New creates a new instance of the store struct.
func New(expiry time.Duration, cleanupInterval time.Duration) *Store {
	store := &Store{
		defaultExpiration: expiry,
		cache:             make(map[string]Value),
	}

	StartCleaner(store, cleanupInterval)
	return store
}

// Set adds a new key/value pair to cache.
// This fails by returning false if the key already exists in cache.
func (store *Store) Set(key string, value interface{}, expires time.Duration) bool {
	store.mutex.RLock()

	_, found := store.cache[key]
	if found {
		return false
	}

	store.mutex.RUnlock()
	store.Put(key, value, expires)
	return true
}

// Put adds or updates a key/value pair.
func (store *Store) Put(key string, value interface{}, expires time.Duration) {
	store.mutex.Lock()
	store.cache[key] = Value{expiry: store.getUnix(expires), Object: value}
	store.mutex.Unlock()
	if store.adapterExists() {
		store.redis.SetToRedis(key, value, expires)
	}
	return
}

func (store *Store) putFromPersistence(key string, value interface{}, expires time.Duration) {
	store.mutex.Lock()
	store.cache[key] = Value{expiry: store.getUnix(expires), Object: value}
	store.mutex.Unlock()
	return
}

// Get retrieves value assigned to key from the store
func (store *Store) Get(key string) (interface{}, bool) {
	store.mutex.RLock()
	value, found := store.cache[key]
	store.mutex.RUnlock()

	if !found {
		// check redis cache for data
		if store.adapterExists() {
			if value, found := store.redis.GetFromRedis(key); found {
				store.putFromPersistence(key, value, DefaultExpiry)
				return value, true
			}
		}
		return nil, false
	}

	if hasExpired(value.expiry) {
		delete(store.cache, key)
		if store.onDeleted != nil {
			store.onDeleted(key, value.Object)
		}
		return nil, false
	}

	return value.Object, true
}

func (store *Store) Delete(key string) {
	if value, found := store.Get(key); found {
		store.mutex.Lock()
		delete(store.cache, key)
		store.mutex.Unlock()
		if store.adapterExists() {
			store.redis.DeleteFromRedis(key)
		}

		if store.onDeleted != nil {
			store.onDeleted(key, value)
		}
	}

	return
}

// OnDeleted callback is called after data is deleted from cache
func (store *Store) OnDelete(fn func(string, interface{})) {
	store.onDeleted = fn
	return
}

// Helper functions
func (store *Store) getUnix(duration time.Duration) int64 {
	if duration == DefaultExpiry {
		return time.Now().Add(store.defaultExpiration).UnixNano()
	}

	if duration == ExpiryInfinity {
		return int64(duration)
	}

	return time.Now().Add(duration).UnixNano()
}

func hasExpired(expires int64) bool {
	// if expiry is infinity, always return false.
	if expires == int64(ExpiryInfinity) {
		return false
	}
	// compare current unix with expiry int
	// if current greater, it has expired, run OnExpired function
	return time.Now().UnixNano() > expires
}

// cleanup run background job for removing expired entries.
// The cleanup service can be in one of `clean` `skip` or `stop`
type cleaner struct {
	cleanupInterval time.Duration
	done            chan bool
}

func (store *Store) Run() {
	ticker := time.NewTicker(store.janitor.cleanupInterval)
	for {
		select {
		case <-ticker.C:
			store.DeleteExpired()

		case status := <-store.janitor.done:
			if status {
				ticker.Stop()
				return
			}
		}
	}
}

func (store *Store) StopCleaner() {
	store.janitor.done <- true
}

func StartCleaner(store *Store, interval time.Duration) {
	store.janitor = &cleaner{
		cleanupInterval: interval,
		done:            make(chan bool),
	}
	go store.Run()
	return
}

func (store *Store) DeleteExpired() {
	// create a map to temporary hold expired values
	var expired_store map[string]Value
	expired_store = make(map[string]Value)

	store.mutex.Lock()
	for key, value := range store.cache {
		if hasExpired(value.expiry) {
			delete(store.cache, key)
			expired_store[key] = value
		}
	}
	store.mutex.Unlock()

	// send now send onDeleted events
	for key, value := range expired_store {
		if store.onDeleted != nil {
			store.onDeleted(key, value)
		}
	}
	expired_store = make(map[string]Value)
}

// Adapter Configruation
func (store *Store) ConfigureAdapter(adapter string, params interface{}, encoder func(string, interface{}) (string, error), decoder func(string, string) (interface{}, error)) *Store {
	switch adapter {
	case "redis":
		if redis_params, ok := params.(*redis.Options); ok {
			store.redis = SetupRedis(redis_params, encoder, decoder)
			return store
		} else {
			panic("Unable to convert params to redis.Options")
		}

	default:
		return nil
	}
}

func (store *Store) adapterExists() bool {
	return store.redis != nil
}
