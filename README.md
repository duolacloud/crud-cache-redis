# CRUD-Core 的 Redis 缓存插件

为 [crud-core](https://github.com/duolacloud/crud-core) 提供了基于 redis 的缓存实现。

## 安装

依赖 `go >= 1.18` ，初始化 go module 后直接安装

```bash
go get github.com/duolacloud/crud-cache-redis
```

## 使用

```go

import "github.com/duolacloud/crud-cache-redis"

// 创建缓存
c, err := cache.NewRedisCache(
	// 设置缓存键前缀
	WithPrefix("APP_CACHE_PREFIX:"),

	// 设置序列化和反序列化，默认是 json.Marshal / json.Unmarshal
	WithMarshal(xml.Marshal),
	WithUnmarshal(xml.Unmarshal),

	// redis 连接配置
	// 设置 redis 连接地址
	WithHost("127.0.0.1:6379"),
	// 设置 redis AUTH 认证
	WithPassword("secret"),
	// 设置 redis SELECT 选择 db
	WithDB(1),
	// 设置 redis 连接池
	WithPoolOptions(5, 20, 30 * time.Minitue),

	// 设置 redis 连接池
	// 设置后会忽略上面提供的 redis 连接配置，直接使用用户提供的连接池
	WithPool(redisPool),
)

// 设置缓存
c.Set(context.TODO(), "key", user, cache.WithExpiration(10 * time.Second))

// 查询缓存
user := &User{}
err := c.Get(context.TODO(), "key", user)

```
