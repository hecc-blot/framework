package cache

type ICacheFactory interface {
	Redis() IRedisCache
	Local() ILocalCache
}
