# Bufrex
Multi Layered key/value system for golang.
Bufrex can be used purely as an in-memory key/value store. It can also be extended to use a Redis adapter to persist cached data into redis database. In memory, bufrex is a thread-safe map with expiration date. When used with redis, the data is first stored in memory and then pushed to redis, making it available across multiple instances connected to the redis instance.

## Benefits
- Persistence in case of crashes or application restarts
- Make local cache available across the cluster of isolated instances.

## Usage 
To use `Bufrex` as a purely local caching system 

```golang
const (
  defaultExpiry = 2*time.Second
  cacheCleanupTime = 5*time.Minute
)

// create a new store.
store := bufrex.New(defaultExpiry, cacheCleanupTime) 

// Set a new key, fails if key already exists 
store.Set("some_key", "hello world", bufrex.DefaultExpiry)

// Put a key, updates if key already exists 
store.Set("some_other_key", "hello world again", 3*time.Minute)

// Return a key 
if value, found := store.Get("some_key"); found {
  fmt.Println(value)
}

// delete a key
store.Delete("some_other_key")
```

When using `Bufrex` with a persistent adapter (currently supports only `redis`)
```golang
// only change happens during the creation of a new store struct 

// anonymous function to send a key string and a value interface.
// the resulting value is a string. Eg. Encoded json string
encoder := func(key string, value interface{}) (string, error) {
  return value.(string), nil
}

// anonymous function receive an encoded string value with key to return the interface 
decoder := func(_ string, value string) (interface{}, error) {
  return value, nil
}

// redis opts
redis_opts := &redis.Options{
  Addr:     "localhost:6379",
  Password: "",
  DB:       0,
}

store := bufrex.New(bufrex.DefaultExpiry, 2*time.Minute).ConfigureAdapter("redis", redis_opts, encoder, decoder)

// Use the same exposed apis to persist in-memory data to redis database.
```
