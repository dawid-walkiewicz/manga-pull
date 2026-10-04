package plugins

import (
	"sync"
)

type RuntimeRegistry struct {
	runtimes map[string]*PluginRuntime
	mu       sync.RWMutex
}

func NewRuntimeRegistry() *RuntimeRegistry {
	return &RuntimeRegistry{
		runtimes: make(map[string]*PluginRuntime),
	}
}

func (r *RuntimeRegistry) Get(id string) (*PluginRuntime, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	runtime, ok := r.runtimes[id]

	return runtime, ok
}

func (r *RuntimeRegistry) Register(id string, runtime *PluginRuntime) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.runtimes[id] = runtime
}

func (r *RuntimeRegistry) Unregister(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.runtimes, id)
}
