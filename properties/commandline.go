package properties

import (
	"maps"
	"sync"
)

// commandline holds the -P/--properties overrides of the most recently parsed flag set. Flag sets
// may be bound and parsed concurrently, e.g. a server building one command tree per request.
var commandline commandlineStore

type commandlineStore struct {
	lock sync.RWMutex
	m    map[string]string
}

func (c *commandlineStore) get(key string) (string, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	value, ok := c.m[key]
	return value, ok
}

func (c *commandlineStore) snapshot() map[string]string {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return maps.Clone(c.m)
}

func (c *commandlineStore) replace(values map[string]string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.m = maps.Clone(values)
}
