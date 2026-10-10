package plugin

import "sync"

// Registry holds all diary plugins registered at application start.
// It is safe for concurrent use.
type Registry struct {
	mu      sync.RWMutex
	plugins map[string]DiaryPlugin
	order   []string
}

func NewRegistry() *Registry {
	return &Registry{plugins: make(map[string]DiaryPlugin)}
}

// Register adds a plugin, keyed by its metadata slug. Re-registering the same
// slug replaces the implementation but keeps its position in the ordering.
func (r *Registry) Register(p DiaryPlugin) {
	r.mu.Lock()
	defer r.mu.Unlock()

	slug := p.Metadata().Slug
	if _, exists := r.plugins[slug]; !exists {
		r.order = append(r.order, slug)
	}
	r.plugins[slug] = p
}

// Get returns the plugin with the given slug.
func (r *Registry) Get(slug string) (DiaryPlugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.plugins[slug]
	return p, ok
}

// All returns every registered plugin in registration order.
func (r *Registry) All() []DiaryPlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]DiaryPlugin, 0, len(r.order))
	for _, slug := range r.order {
		out = append(out, r.plugins[slug])
	}
	return out
}
