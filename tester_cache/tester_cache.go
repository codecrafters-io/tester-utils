package tester_cache

import (
	"errors"
)

var (
	ErrNotFound = errors.New("Key not found")
)

// cache interface should be satisfied by every underlying cache implementation
type cache interface {
	Get(key string) ([]byte, bool)
	Set(key string, value []byte)
	Close()
}

type TesterCache struct {
	testerCacheImplementation cache
}

func New() *TesterCache {
	redisCache, err := newRedisCache()

	// use zero cache if redis is not available
	if err != nil {
		return &TesterCache{
			testerCacheImplementation: newZeroCache(),
		}
	}

	return &TesterCache{
		testerCacheImplementation: redisCache,
	}
}

func (c *TesterCache) Get(key string) ([]byte, bool) {
	return c.testerCacheImplementation.Get(key)
}

func (c *TesterCache) Set(key string, value []byte) {
	c.testerCacheImplementation.Set(key, value)
}

func (c *TesterCache) Close() {
	c.testerCacheImplementation.Close()
}
