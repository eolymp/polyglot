package oauth

import "time"

type cacheClient interface {
	ShouldSet(key string, value interface{}, ttl time.Duration)
	ShouldGet(key string, value interface{}) (hit bool)
}

type nopCache struct{}

func (nopCache) ShouldSet(key string, value interface{}, ttl time.Duration) {}

func (nopCache) ShouldGet(key string, value interface{}) bool {
	return false
}
