package pongo2

import (
	"container/list"
	"sync"
)

const (
	defaultExecTemplateCacheCapacity     = 16384
	defaultExecTemplateCacheMaxBodyBytes = 1024
)

type execTemplateCacheKey struct {
	set  *TemplateSet
	body string
}

type execTemplateCacheEntry struct {
	key execTemplateCacheKey
	tpl *Template
}

// execTemplateCache is a bounded LRU for templates parsed from rendered exec and
// allowmissingval bodies. Template parsing depends on the TemplateSet (sandboxed
// tags/filters, globals/loaders and registered parse functions), so the key pairs
// the rendered body with the originating set. Option modes that mutate tokens at
// execution time, debug mode and bodies with known per-execution state or
// parse-time loader dependencies bypass this cache in fromBytesCached.
type execTemplateCache struct {
	mu           sync.Mutex
	capacity     int
	maxBodyBytes int
	ll           *list.List // front = most recently used
	entries      map[execTemplateCacheKey]*list.Element
	hits         int64
	misses       int64
	evictions    int64
}

func newExecTemplateCache(capacity, maxBodyBytes int) *execTemplateCache {
	if maxBodyBytes < 0 {
		maxBodyBytes = 0
	}
	return &execTemplateCache{
		capacity:     capacity,
		maxBodyBytes: maxBodyBytes,
		ll:           list.New(),
		entries:      make(map[execTemplateCacheKey]*list.Element),
	}
}

func (c *execTemplateCache) cacheable(body []byte) bool {
	return c.capacity > 0 && len(body) <= c.maxBodyBytes
}

func (c *execTemplateCache) get(key execTemplateCacheKey) (*Template, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if el, ok := c.entries[key]; ok {
		c.ll.MoveToFront(el)
		c.hits++
		return el.Value.(*execTemplateCacheEntry).tpl, true
	}

	c.misses++
	return nil, false
}

func (c *execTemplateCache) recordMiss() {
	c.mu.Lock()
	c.misses++
	c.mu.Unlock()
}

func (c *execTemplateCache) put(key execTemplateCacheKey, tpl *Template) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.capacity <= 0 {
		return
	}

	if _, ok := c.entries[key]; ok {
		// First writer wins for duplicate parses of the same body in the same
		// parse context; concurrent duplicate misses are harmless.
		return
	}

	c.entries[key] = c.ll.PushFront(&execTemplateCacheEntry{key: key, tpl: tpl})
	for c.ll.Len() > c.capacity {
		oldest := c.ll.Back()
		c.ll.Remove(oldest)
		delete(c.entries, oldest.Value.(*execTemplateCacheEntry).key)
		c.evictions++
	}
}

func (c *execTemplateCache) stats() (hits, misses, evictions int64, entries int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.hits, c.misses, c.evictions, c.ll.Len()
}

var (
	execTemplateCacheMu       sync.RWMutex
	execTemplateCacheInstance = newExecTemplateCache(defaultExecTemplateCacheCapacity, defaultExecTemplateCacheMaxBodyBytes)
)

func currentExecTemplateCache() *execTemplateCache {
	execTemplateCacheMu.RLock()
	defer execTemplateCacheMu.RUnlock()

	return execTemplateCacheInstance
}

// SetExecTemplateCache replaces the package-level exec-body template cache and
// resets its hit, miss, eviction and entry counters. A capacity less than or equal
// to zero disables caching, while still counting each attempted lookup as a miss.
func SetExecTemplateCache(capacity, maxBodyBytes int) {
	execTemplateCacheMu.Lock()
	defer execTemplateCacheMu.Unlock()

	execTemplateCacheInstance = newExecTemplateCache(capacity, maxBodyBytes)
}

// ExecTemplateCacheStats reports hit, miss and eviction counters plus the current
// number of cached entries for the active exec-body template cache instance.
func ExecTemplateCacheStats() (hits, misses, evictions int64, entries int) {
	return currentExecTemplateCache().stats()
}

func (set *TemplateSet) fromBytesCached(body []byte) (*Template, error) {
	cache := currentExecTemplateCache()
	if !cache.cacheable(body) || set.execTemplateCacheBypass(body) {
		cache.recordMiss()
		return set.FromBytes(body)
	}

	key := execTemplateCacheKey{set: set, body: string(body)}
	if tpl, ok := cache.get(key); ok {
		return tpl, nil
	}

	tpl, err := set.FromBytes(body)
	if err != nil {
		return nil, err
	}
	cache.put(key, tpl)
	return tpl, nil
}

func (set *TemplateSet) execTemplateCacheBypass(body []byte) bool {
	trimBlocks, lStripBlocks := execTemplateCacheOptions(set)
	if set.Debug || trimBlocks || lStripBlocks {
		return true
	}
	return execTemplateBodyHasBypassTag(body)
}

func execTemplateCacheOptions(set *TemplateSet) (trimBlocks, lStripBlocks bool) {
	if set == nil || set.Options == nil {
		return false, false
	}
	return set.Options.TrimBlocks, set.Options.LStripBlocks
}

func execTemplateBodyHasBypassTag(body []byte) bool {
	tokens, err := lex("<exec-cache>", string(body))
	if err != nil {
		// Parsing will return the actual error. Avoid retaining a possibly malformed
		// body in the shared cache while that happens.
		return true
	}

	for i := 0; i+1 < len(tokens); i++ {
		if tokens[i].Typ == TokenSymbol && tokens[i].Val == "{%" && tokens[i+1].Typ == TokenIdentifier {
			switch tokens[i+1].Val {
			case "cycle", "ifchanged", "include", "extends", "import", "ssi":
				return true
			}
		}
	}
	return false
}
