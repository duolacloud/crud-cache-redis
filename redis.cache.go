package cache

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"

	"github.com/duolacloud/crud-core/cache"
	"github.com/duolacloud/crud-core/types"
	"github.com/redis/go-redis/v9"
)

// 基于 redis 的缓存
type RedisCache struct {
	prefix        string        // 缓存键的前缀
	marshal       MarshalFunc   // 将 struct 序列化为字节数组
	unmarshal     UnmarshalFunc // 将字节数组反序列化为 struct
	addr          string        // redis连接
	password      string        // redis 认证密码
	db            int           // redis 选择的 db
	client        *redis.Client // redis 连接实例
	clientOptions *redis.Options
	// clusterClient  *redis.ClusterClient
	// clusterOptions *redis.ClusterOptions
	tls bool
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
func WithAddr(addr string) Option {
	return func(rc *RedisCache) {
		rc.addr = addr
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

// 缓存将使用此client，而不是自己创建
func WithClient(client *redis.Client) Option {
	return func(rc *RedisCache) {
		rc.client = client
	}
}

// 设置 redis 选择的 db
func WithTLS(tls bool) Option {
	return func(rc *RedisCache) {
		rc.tls = tls
	}
}

func WithClientOptions(clientOptions *redis.Options) Option {
	return func(rc *RedisCache) {
		rc.clientOptions = clientOptions
	}
}

/*
func WithClusterOptions(clusterOptions *redis.ClusterOptions) Option {
	return func(rc *RedisCache) {
		rc.clusterOptions = clusterOptions
	}
}
*/

func New(opts ...Option) (cache.Cache, error) {
	c := &RedisCache{
		addr:      "localhost:6379",
		marshal:   json.Marshal,
		unmarshal: json.Unmarshal,
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.client == nil {
		c.newClient()
	}
	return c, nil
}

func (rc *RedisCache) newClient() {
	options := rc.clientOptions
	if options == nil {
		options = &redis.Options{}
	}

	if len(rc.addr) > 0 {
		options.Addr = rc.addr
	}

	if len(rc.password) > 0 {
		options.Password = rc.password
	}

	if rc.db != 0 {
		options.DB = rc.db
	}

	if rc.tls {
		options.TLSConfig = &tls.Config{}
	}

	rc.client = redis.NewClient(options)
}

func (rc *RedisCache) Get(ctx context.Context, key string, value any, opts ...cache.GetOption) error {
	options := &cache.GetOptions{}
	for _, opt := range opts {
		opt(options)
	}

	cacheKey := rc.prefix + key
	bytes, err := rc.client.Get(ctx, cacheKey).Bytes()
	if err != nil {
		return wrapRedisError(err)
	}

	return rc.unmarshal(bytes, &value)
}

func (rc *RedisCache) Set(ctx context.Context, key string, value any, opts ...cache.SetOption) error {
	options := &cache.SetOptions{}
	for _, opt := range opts {
		opt(options)
	}
	bytes, err := rc.marshal(value)
	if err != nil {
		return err
	}

	cacheKey := rc.prefix + key
	err = rc.client.Set(ctx, cacheKey, bytes, options.Exipration).Err()

	return err
}

func (rc *RedisCache) Delete(ctx context.Context, key string, opts ...cache.DeleteOption) error {
	options := &cache.DeleteOptions{}
	for _, opt := range opts {
		opt(options)
	}

	cacheKey := rc.prefix + key
	err := rc.client.Del(ctx, cacheKey).Err()
	return err
}

func wrapRedisError(err error) error {
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return types.ErrNotFound
		}
	}
	return err
}
