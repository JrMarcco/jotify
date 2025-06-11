//go:build e2e

package idempotent

import (
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisStrategy_Exists(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	client := redis.NewClient(&redis.Options{
		Addr:     "192.168.3.3:6379",
		Password: "<passwd>",
	})
	defer func() {
		client.Del(ctx, "test_key", "test_new_key")
		_ = client.Close()
	}()

	strategy := NewRedisStrategy(client, time.Second)

	key := "test_key"
	exists, err := strategy.Exists(ctx, key)
	require.NoError(t, err)
	assert.False(t, exists)

	exists, err = strategy.Exists(ctx, key)
	require.NoError(t, err)
	assert.True(t, exists)

	newKey := "test_new_key"
	exists, err = strategy.Exists(ctx, newKey)
	require.NoError(t, err)
	assert.False(t, exists)

	time.Sleep(time.Second)
	exists, err = strategy.Exists(ctx, key)
	require.NoError(t, err)
	assert.False(t, exists)

}

func TestRedisStrategy_MultiExists(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	client := redis.NewClient(&redis.Options{
		Addr:     "192.168.3.3:6379",
		Password: "<passwd>",
	})

	keySize := 100
	keys := make([]string, keySize)

	for i := 0; i < keySize; i++ {
		keys[i] = fmt.Sprintf("test_key_%d", i)
	}

	defer func() {
		client.Del(ctx, keys...)
		_ = client.Close()
	}()

	strategy := NewRedisStrategy(client, 5*time.Second)

	res, err := strategy.MultiExists(ctx, keys)
	require.NoError(t, err)
	assert.Equal(t, keySize, len(res))
	for _, key := range keys {
		keyExists, ok := res[key]
		assert.True(t, ok)
		assert.False(t, keyExists)
	}

	res, err = strategy.MultiExists(ctx, keys)
	require.NoError(t, err)
	assert.Equal(t, keySize, len(res))
	for _, key := range keys {
		keyExists, ok := res[key]
		assert.True(t, ok)
		assert.True(t, keyExists)
	}

	time.Sleep(5 * time.Second)

	_, err = strategy.Exists(ctx, "test_key_1")
	_, err = strategy.Exists(ctx, "test_key_10")

	res, err = strategy.MultiExists(ctx, keys)
	require.NoError(t, err)
	assert.Equal(t, keySize, len(res))
	for _, key := range keys {
		keyExists, ok := res[key]
		assert.True(t, ok)

		if key == "test_key_1" || key == "test_key_10" {
			assert.True(t, keyExists)
			continue
		}
		assert.False(t, keyExists)
	}

	res, err = strategy.MultiExists(ctx, keys)
	require.NoError(t, err)
	assert.Equal(t, keySize, len(res))
	for _, key := range keys {
		keyExists, ok := res[key]
		assert.True(t, ok)
		assert.True(t, keyExists)
	}
}
