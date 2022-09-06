package cache

import (
	"context"
	"testing"
	"time"

	"github.com/duolacloud/crud-core/cache"
	"github.com/stretchr/testify/assert"
)

type User struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestRedisCache(t *testing.T) {
	redisCache, err := NewRedisCache(WithPrefix("curd-cache-redis:"))
	assert.Nil(t, err)

	user1 := &User{
		Name: "jack",
		Age:  18,
	}
	err = redisCache.Set(context.TODO(), "test_key1", user1, cache.WithExpiration(5*time.Second))
	assert.Nil(t, err)

	foundUser1 := new(User)
	err = redisCache.Get(context.TODO(), "test_key1", foundUser1)
	assert.Nil(t, err)
	assert.Equal(t, user1.Name, foundUser1.Name)
	assert.Equal(t, user1.Age, foundUser1.Age)

	time.Sleep(6 * time.Second)
	err = redisCache.Get(context.TODO(), "test_key1", foundUser1)
	assert.Same(t, err, ErrNotExist)

	user2 := &User{
		Name: "rose",
		Age:  20,
	}
	err = redisCache.Set(context.TODO(), "test_key2", user2)
	assert.Nil(t, err)

	foundUser2 := new(User)
	err = redisCache.Get(context.TODO(), "test_key2", foundUser2)
	assert.Nil(t, err)
	assert.Equal(t, foundUser2.Name, user2.Name)

	err = redisCache.Delete(context.TODO(), "test_key2")
	assert.Nil(t, err)

	err = redisCache.Get(context.TODO(), "test_key2", foundUser2)
	assert.Same(t, err, ErrNotExist)
}
