package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/duolacloud/crud-core/cache"
	"github.com/gomodule/redigo/redis"
)

type RedisCache struct {
	host        string
	prefix      string
	marshal     MarshalFunc
	unmarshal   UnmarshalFunc
	password    string
	maxIdle     int
	maxActive   int
	idleTimeout time.Duration
	db          int
	redisPool   *redis.Pool
}

type MarshalFunc func(any) ([]byte, error)
type UnmarshalFunc func([]byte, any) error

type Option func(*RedisCache)

func WithHost(host string) Option {
	return func(rc *RedisCache) {
		rc.host = host
	}
}

func WithPrefix(prefix string) Option {
	return func(rc *RedisCache) {
		rc.prefix = prefix
	}
}

func WithMarshal(marshal MarshalFunc) Option {
	return func(rc *RedisCache) {
		rc.marshal = marshal
	}
}

func WithUnmarshal(unmarshal UnmarshalFunc) Option {
	return func(rc *RedisCache) {
		rc.unmarshal = unmarshal
	}
}

func WithPassword(password string) Option {
	return func(rc *RedisCache) {
		rc.password = password
	}
}

func WithPool(maxIdle, maxActive int, idleTimeout time.Duration) Option {
	return func(rc *RedisCache) {
		rc.maxIdle = maxIdle
		rc.maxActive = maxActive
		rc.idleTimeout = idleTimeout
	}
}

func WithDB(db int) Option {
	return func(rc *RedisCache) {
		rc.db = db
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
	c.connect()
	return c, nil
}

func (rc *RedisCache) connect() {
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
		if redis.ErrNil == err {
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
