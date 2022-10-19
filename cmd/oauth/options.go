package oauth

import (
	"github.com/eolymp/go-packages/httpx"
	"time"
)

type Option func(*Client)

// WithClient sets HTTP client used to communicate with OAuth server
func WithClient(client httpx.Client) Option {
	return func(c *Client) {
		c.client = client
	}
}

// WithCache adds cache for token introspection request
func WithCache(cache cacheClient) Option {
	return func(c *Client) {
		c.cache = cache
	}
}

// WithCacheTTL sets cache TTL for token introspection request
func WithCacheTTL(ttl time.Duration) Option {
	return func(c *Client) {
		c.cacheTTL = ttl
	}
}
