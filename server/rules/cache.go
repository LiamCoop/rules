package rules

import "time"

// RulesCache provides an abstraction for caching active rules
// This allows swapping between in-memory, Redis, or other caching implementations
type RulesCache interface {
	// Get retrieves cached rules, returns nil if cache miss or expired
	Get() []*Rule

	// Set stores rules in cache
	Set(rules []*Rule)

	// Invalidate clears the cache, forcing a refresh on next Get
	Invalidate()

	// IsValid returns true if cache has valid data
	IsValid() bool
}

// CacheConfig holds configuration for cache behavior
type CacheConfig struct {
	// TTL is the time-to-live for cached entries
	// Set to 0 for no expiration (manual invalidation only)
	TTL time.Duration

	// RefreshOnInvalidate determines if cache should be refreshed immediately
	// when invalidated, or wait for next Get call
	RefreshOnInvalidate bool
}

// DefaultCacheConfig returns sensible defaults for rule caching
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		TTL:                 0, // No TTL - only invalidate on mutations
		RefreshOnInvalidate: false,
	}
}
