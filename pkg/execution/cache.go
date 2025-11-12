package execution

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/dshills/goflow/pkg/domain/execution"
	"github.com/dshills/goflow/pkg/domain/types"
)

// CacheEntry represents a cached node execution result
type CacheEntry struct {
	NodeID      types.NodeID
	NodeType    string
	InputsHash  string
	Outputs     map[string]interface{}
	CachedAt    time.Time
	AccessCount int64
	LastAccess  time.Time
}

// CacheStats tracks cache performance metrics
type CacheStats struct {
	Hits        int64
	Misses      int64
	Evictions   int64
	TotalSize   int64
	HitRate     float64
	LastUpdated time.Time
}

// ExecutionCache provides caching for node execution results
// It caches outputs based on node type and input hashes, allowing
// skipping of unchanged nodes during workflow execution
type ExecutionCache struct {
	entries map[string]*CacheEntry // key: nodeID + inputsHash
	stats   *CacheStats
	mu      sync.RWMutex
	maxSize int
	ttl     time.Duration
	enabled bool
}

// NewExecutionCache creates a new execution cache with default settings
func NewExecutionCache() *ExecutionCache {
	return &ExecutionCache{
		entries: make(map[string]*CacheEntry),
		stats: &CacheStats{
			LastUpdated: time.Now(),
		},
		maxSize: 1000,             // Cache up to 1000 entries by default
		ttl:     30 * time.Minute, // Default TTL of 30 minutes
		enabled: true,
	}
}

// NewExecutionCacheWithConfig creates a cache with custom configuration
func NewExecutionCacheWithConfig(maxSize int, ttl time.Duration) *ExecutionCache {
	return &ExecutionCache{
		entries: make(map[string]*CacheEntry),
		stats: &CacheStats{
			LastUpdated: time.Now(),
		},
		maxSize: maxSize,
		ttl:     ttl,
		enabled: true,
	}
}

// Enable enables the cache
func (c *ExecutionCache) Enable() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enabled = true
}

// Disable disables the cache
func (c *ExecutionCache) Disable() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enabled = false
}

// IsEnabled returns whether the cache is enabled
func (c *ExecutionCache) IsEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enabled
}

// Get retrieves a cached result if available and still valid
func (c *ExecutionCache) Get(nodeID types.NodeID, nodeType string, inputs map[string]interface{}) (*CacheEntry, bool) {
	if !c.IsEnabled() {
		return nil, false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	// Compute input hash
	inputsHash, err := c.hashInputs(inputs)
	if err != nil {
		return nil, false
	}

	// Build cache key
	key := c.buildKey(nodeID, inputsHash)

	// Look up entry
	entry, exists := c.entries[key]
	if !exists {
		c.incrementMisses()
		return nil, false
	}

	// Check if entry has expired
	if time.Since(entry.CachedAt) > c.ttl {
		c.incrementMisses()
		return nil, false
	}

	// Update access statistics
	c.incrementHits()
	entry.AccessCount++
	entry.LastAccess = time.Now()

	// Return a copy to prevent mutation
	entryCopy := &CacheEntry{
		NodeID:      entry.NodeID,
		NodeType:    entry.NodeType,
		InputsHash:  entry.InputsHash,
		Outputs:     c.deepCopyOutputs(entry.Outputs),
		CachedAt:    entry.CachedAt,
		AccessCount: entry.AccessCount,
		LastAccess:  entry.LastAccess,
	}

	return entryCopy, true
}

// Set stores a node execution result in the cache
func (c *ExecutionCache) Set(nodeID types.NodeID, nodeType string, inputs map[string]interface{}, outputs map[string]interface{}) error {
	if !c.IsEnabled() {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Compute input hash
	inputsHash, err := c.hashInputs(inputs)
	if err != nil {
		return fmt.Errorf("failed to hash inputs: %w", err)
	}

	// Build cache key
	key := c.buildKey(nodeID, inputsHash)

	// Check if we need to evict entries
	if len(c.entries) >= c.maxSize {
		c.evictOldest()
	}

	// Create and store entry
	entry := &CacheEntry{
		NodeID:      nodeID,
		NodeType:    nodeType,
		InputsHash:  inputsHash,
		Outputs:     c.deepCopyOutputs(outputs),
		CachedAt:    time.Now(),
		AccessCount: 0,
		LastAccess:  time.Now(),
	}

	c.entries[key] = entry
	c.stats.TotalSize = int64(len(c.entries))
	c.stats.LastUpdated = time.Now()

	return nil
}

// Invalidate removes a specific cache entry
func (c *ExecutionCache) Invalidate(nodeID types.NodeID, inputs map[string]interface{}) error {
	if !c.IsEnabled() {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Compute input hash
	inputsHash, err := c.hashInputs(inputs)
	if err != nil {
		return fmt.Errorf("failed to hash inputs: %w", err)
	}

	// Build cache key
	key := c.buildKey(nodeID, inputsHash)

	// Remove entry if it exists
	delete(c.entries, key)
	c.stats.TotalSize = int64(len(c.entries))
	c.stats.LastUpdated = time.Now()

	return nil
}

// InvalidateNode removes all cache entries for a specific node
func (c *ExecutionCache) InvalidateNode(nodeID types.NodeID) {
	if !c.IsEnabled() {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Find and remove all entries for this node
	for key, entry := range c.entries {
		if entry.NodeID == nodeID {
			delete(c.entries, key)
		}
	}

	c.stats.TotalSize = int64(len(c.entries))
	c.stats.LastUpdated = time.Now()
}

// Clear removes all entries from the cache
func (c *ExecutionCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*CacheEntry)
	c.stats.TotalSize = 0
	c.stats.LastUpdated = time.Now()
}

// GetStats returns current cache statistics
func (c *ExecutionCache) GetStats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := *c.stats
	stats.TotalSize = int64(len(c.entries))

	// Calculate hit rate
	total := stats.Hits + stats.Misses
	if total > 0 {
		stats.HitRate = float64(stats.Hits) / float64(total)
	}

	return stats
}

// CleanExpired removes all expired entries
func (c *ExecutionCache) CleanExpired() int {
	if !c.IsEnabled() {
		return 0
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	removed := 0

	for key, entry := range c.entries {
		if now.Sub(entry.CachedAt) > c.ttl {
			delete(c.entries, key)
			removed++
		}
	}

	if removed > 0 {
		c.stats.Evictions += int64(removed)
		c.stats.TotalSize = int64(len(c.entries))
		c.stats.LastUpdated = time.Now()
	}

	return removed
}

// ShouldCache determines if a node execution result should be cached
// based on node type and execution characteristics
func (c *ExecutionCache) ShouldCache(nodeExec *execution.NodeExecution) bool {
	if !c.IsEnabled() {
		return false
	}

	// Don't cache failed executions
	if nodeExec.Status == execution.NodeStatusFailed {
		return false
	}

	// Don't cache start/end nodes (they're trivial)
	if nodeExec.NodeType == "start" || nodeExec.NodeType == "end" {
		return false
	}

	// Don't cache nodes with no outputs
	if len(nodeExec.Outputs) == 0 {
		return false
	}

	// Cache MCP tool calls, transforms, and conditions
	// These are deterministic and safe to cache
	switch nodeExec.NodeType {
	case "mcp_tool", "transform", "condition":
		return true
	default:
		// Don't cache parallel or loop nodes (they're complex)
		return false
	}
}

// buildKey creates a cache key from node ID and input hash
func (c *ExecutionCache) buildKey(nodeID types.NodeID, inputsHash string) string {
	return fmt.Sprintf("%s:%s", nodeID, inputsHash)
}

// hashInputs creates a deterministic hash of input parameters
func (c *ExecutionCache) hashInputs(inputs map[string]interface{}) (string, error) {
	// Convert inputs to JSON for consistent hashing
	jsonBytes, err := json.Marshal(inputs)
	if err != nil {
		return "", err
	}

	// Compute SHA-256 hash
	hash := sha256.Sum256(jsonBytes)
	return fmt.Sprintf("%x", hash), nil
}

// deepCopyOutputs creates a deep copy of outputs to prevent mutation
func (c *ExecutionCache) deepCopyOutputs(outputs map[string]interface{}) map[string]interface{} {
	if outputs == nil {
		return nil
	}

	// Use JSON marshal/unmarshal for deep copy
	// This works for JSON-serializable types
	jsonBytes, err := json.Marshal(outputs)
	if err != nil {
		return outputs // Return original on error
	}

	var copied map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &copied); err != nil {
		return outputs // Return original on error
	}

	return copied
}

// evictOldest removes the least recently used entry
func (c *ExecutionCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	// Find the entry with the oldest access time
	for key, entry := range c.entries {
		if oldestKey == "" || entry.LastAccess.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.LastAccess
		}
	}

	// Remove the oldest entry
	if oldestKey != "" {
		delete(c.entries, oldestKey)
		c.stats.Evictions++
	}
}

// incrementHits atomically increments the hit counter
func (c *ExecutionCache) incrementHits() {
	c.stats.Hits++
	c.stats.LastUpdated = time.Now()
}

// incrementMisses atomically increments the miss counter
func (c *ExecutionCache) incrementMisses() {
	c.stats.Misses++
	c.stats.LastUpdated = time.Now()
}
