package bufrex

import (
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestSetupAdapter(t *testing.T) {
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

	adapter := SetupRedis(redis_opts, encoder, decoder)
	if adapter.client == nil || adapter.onDecode == nil || adapter.onEncode == nil {
		t.Fatal("Unknown data parsed")
	}
}

func TestSetToRedisGetDelete(t *testing.T) {
	adapter := setupRedis()
	adapter.SetToRedis("some_key", "hello world", 200*time.Millisecond)
	data, found := adapter.GetFromRedis("some_key")
	if !found {
		t.Fatal("inserted data `some_key` not found")
	}

	if data.(string) != "hello world" {
		t.Fatal(data)
	}
	time.Sleep(300 * time.Millisecond)
	if data, found := adapter.GetFromRedis("some_key"); found || data != nil {
		t.Fatal("Data should have been deleted from redis")
	}
}

type TestPayload struct {
	age  int
	name string
}

func TestSetGetDeleteRedisFail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("No panic occured!")
		}
	}()

	adapter := setupRedis()
	value := TestPayload{name: "github", age: 31}
	adapter.SetToRedis("some_other_key", value, 200*time.Millisecond)
}

func TestDeleteNonExisting(t *testing.T) {
	adapter := setupRedis()
	exists := adapter.DeleteFromRedis("not existing")
	if exists {
		t.Fatal("Data was not created")
	}
}

func TestPanicSetupRedis(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Panic was not received")
		}
	}()
	encoder := func(_ string, value interface{}) (string, error) {
		return value.(string), nil
	}

	decoder := func(_ string, value string) (interface{}, error) {
		return value, nil
	}

	redis_opts := &redis.Options{
		Addr:     "localhost:639",
		Password: "",
		DB:       0,
	}
	SetupRedis(redis_opts, encoder, decoder)
}

func TestEncoderPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Panic for not failed encoder was not received")
		}
	}()
	encoder := func(_ string, value interface{}) (string, error) {
		return value.(string), errors.New("Encoder failed to encode")
	}

	decoder := func(_ string, value string) (interface{}, error) {
		return value, nil
	}

	redis_opts := &redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	}
	redis := SetupRedis(redis_opts, encoder, decoder)
	redis.SetToRedis("redis_failure", "hello set faiilure", 2*time.Second)
}

func TestDecoderPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Panic for not failed decode was not received")
		}
	}()
	encoder := func(_ string, value interface{}) (string, error) {
		return value.(string), nil
	}

	decoder := func(_ string, value string) (interface{}, error) {
		return value, errors.New("Decoder failed to decode")
	}

	redis_opts := &redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	}
	redis := SetupRedis(redis_opts, encoder, decoder)
	redis.SetToRedis("redis_decode", "hello world", 2*time.Second)
	redis.GetFromRedis("redis_decode")
}

func setupRedis() *RedisAdapter {
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
	return SetupRedis(redis_opts, encoder, decoder)
}
