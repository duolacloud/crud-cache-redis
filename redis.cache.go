package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/duolacloud/crud-core/cache"
	"github.com/gomodule/redigo/redis"
)

// 基于 redis 的缓存
type RedisCache struct {
	prefix      string        // 缓存键的前缀
	marshal     MarshalFunc   // 将 struct 序列化为字节数组
	unmarshal   UnmarshalFunc // 将字节数组反序列化为 struct
	host        string        // redis连接
	password    string        // redis 认证密码
	maxIdle     int           // redis 连接池最大空闲连接
	maxActive   int           // redis 连接池最大连接数
	idleTimeout time.Duration // redis 连接池空闲超时时间，超时后连接被回收
	db          int           // redis 选择的 db
	redisPool   *redis.Pool   // redis 连接池实例
}

type MarshalFunc func(any) ([]byte, error)
type UnmarshalFunc func([]byte, any) error

type Option func(*RedisCache)

// 设置缓存键的前缀
func WithPrefix(prefix string) Option {
	return func(rc *RedisCache) {
		rc.prefix = prefix
	}
}

// 设置序列化函数
func WithMarshal(marshal MarshalFunc) Option {
	return func(rc *RedisCache) {
		rc.marshal = marshal
	}
}

// 设置反序列化函数
func WithUnmarshal(unmarshal UnmarshalFunc) Option {
	return func(rc *RedisCache) {
		rc.unmarshal = unmarshal
	}
}

// 设置 redis 连接地址
func WithHost(host string) Option {
	return func(rc *RedisCache) {
		rc.host = host
	}
}

// 设置 redis 认证密码
func WithPassword(password string) Option {
	return func(rc *RedisCache) {
		rc.password = password
	}
}

// 设置 redis 选择的 db
func WithDB(db int) Option {
	return func(rc *RedisCache) {
		rc.db = db
	}
}

// 设置连接池，缓存将使用此连接池，而不是自己创建
func WithPool(redisPool *redis.Pool) Option {
	return func(rc *RedisCache) {
		rc.redisPool = redisPool
	}
}

// 设置连接池配置
func WithPoolOptions(maxIdle, maxActive int, idleTimeout time.Duration) Option {
	return func(rc *RedisCache) {
		rc.maxIdle = maxIdle
		rc.maxActive = maxActive
		rc.idleTimeout = idleTimeout
	}
}

func NewRedisCache(opts ...Option) (cache.Cache, error) {
	c := &RedisCache{
		host:        "localhost:6379",
		marshal:     json.Marshal,
		unmarshal:   json.Unmarshal,
		maxIdle:     5,
		maxActive:   20,
		idleTimeout: 10 * time.Minute,
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.redisPool == nil {
		c.newPool()
	}
	return c, nil
}

func (rc *RedisCache) newPool() {
	rc.redisPool = &redis.Pool{
		MaxIdle:     rc.maxIdle,
		MaxActive:   rc.maxActive,
		IdleTimeout: rc.idleTimeout,
		Wait:        true,
		Dial: func() (redis.Conn, error) {
			conn, err := redis.Dial("tcp", rc.host)
			if err != nil {
				return nil, err
			}
			if rc.db > 0 {
				if _, err = conn.Do("SELECT", rc.db); err != nil {
					conn.Close()
					return nil, err
				}
			}
			if rc.password != "" {
				if _, err := conn.Do("AUTH", rc.password); err != nil {
					conn.Close()
					return nil, err
				}
			}
			return conn, nil
		},
		TestOnBorrow: func(conn redis.Conn, t time.Time) error {
			if _, err := conn.Do("PING"); err != nil {
				return err
			}
			return nil
		}}
}

func (rc *RedisCache) Get(c context.Context, key string, value any, opts ...cache.GetOption) error {
	options := &cache.GetOptions{}
	for _, opt := range opts {
		opt(options)
	}
	cacheKey := rc.prefix + key
	bytes, err := redis.Bytes(rc.redisPool.Get().Do("GET", cacheKey))
	if err != nil {
		if errors.Is(err, redis.ErrNil) {
			return cache.ErrNotExsit
		} else {
			return err
		}
	}
	if bytes == nil {
		return nil
	}
	return rc.unmarshal(bytes, value)
}

func (rc *RedisCache) Set(c context.Context, key string, value any, opts ...cache.SetOption) error {
	options := &cache.SetOptions{}
	for _, opt := range opts {
		opt(options)
	}
	bytes, err := rc.marshal(value)
	if err != nil {
		return err
	}
	cacheKey := rc.prefix + key
	expiresIn := options.Exipration.Seconds()
	if expiresIn > 0 {
		_, err = rc.redisPool.Get().Do("SETEX", cacheKey, expiresIn, bytes)
	} else {
		_, err = rc.redisPool.Get().Do("SET", cacheKey, bytes)
	}
	return err
}

func (rc *RedisCache) Delete(c context.Context, key string, opts ...cache.DeleteOption) error {
	options := &cache.DeleteOptions{}
	for _, opt := range opts {
		opt(options)
	}
	cacheKey := rc.prefix + key
	_, err := rc.redisPool.Get().Do("DEL", cacheKey)
	return err
}
