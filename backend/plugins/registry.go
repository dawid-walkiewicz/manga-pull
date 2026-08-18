package plugins

type RuntimeRegistry struct {
	runtimes map[string]*PluginRuntime
}

func NewRuntimeRegistry() *RuntimeRegistry {
	return &RuntimeRegistry{
		runtimes: make(map[string]*PluginRuntime),
	}
}

func (r *RuntimeRegistry) Get(id string) (*PluginRuntime, bool) {
	runtime, ok := r.runtimes[id]

	return runtime, ok
}

func (r *RuntimeRegistry) Register(runtime *PluginRuntime) {
	r.runtimes[runtime.Plugin.ID] = runtime
}

func (r *RuntimeRegistry) Unregister(id string) {
	delete(r.runtimes, id)
}
