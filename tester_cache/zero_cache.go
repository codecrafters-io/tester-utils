package tester_cache

// zeroCache is an empty cache implementation
type zeroCache struct {
}

func newZeroCache() *zeroCache {
	return &zeroCache{}
}

func (c *zeroCache) Get(key string) ([]byte, bool) {
	return nil, false
}

func (c *zeroCache) Set(key string, value []byte) {
	return
}

func (z *zeroCache) Close() {
	return
}
