package rules

import (
	"sync"
	"time"
)

// InMemoryRulesCache is a simple in-memory implementation of RulesCache
// Thread-safe for concurrent access
type InMemoryRulesCache struct {
	rules     []*Rule
	cachedAt  time.Time
	config    CacheConfig
	mu        sync.RWMutex
	isValid   bool
}

// NewInMemoryRulesCache creates a new in-memory rules cache
func NewInMemoryRulesCache(config CacheConfig) *InMemoryRulesCache {
	return &InMemoryRulesCache{
		config:  config,
		isValid: false,
	}
}

// Get retrieves cached rules
// Returns nil if cache is invalid or expired
func (c *InMemoryRulesCache) Get() []*Rule {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check if cache is valid
	if !c.isValid {
		return nil
	}

	// Check TTL if configured
	if c.config.TTL > 0 {
		if time.Since(c.cachedAt) > c.config.TTL {
			// Cache expired
			return nil
		}
	}

	// Return copy to prevent external modifications
	rulesCopy := make([]*Rule, len(c.rules))
	copy(rulesCopy, c.rules)
	return rulesCopy
}

// Set stores rules in cache
func (c *InMemoryRulesCache) Set(rules []*Rule) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Store copy to prevent external modifications
	c.rules = make([]*Rule, len(rules))
	copy(c.rules, rules)
	c.cachedAt = time.Now()
	c.isValid = true
}

// Invalidate clears the cache
func (c *InMemoryRulesCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.isValid = false
	c.rules = nil
}

// IsValid returns true if cache contains valid data
func (c *InMemoryRulesCache) IsValid() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.isValid {
		return false
	}

	// Check TTL if configured
	if c.config.TTL > 0 {
		return time.Since(c.cachedAt) <= c.config.TTL
	}

	return true
}
