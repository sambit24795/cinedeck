package indexer

// Registry holds the known providers in registration order.
type Registry struct {
	providers []Provider
	byName    map[string]Provider
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{byName: make(map[string]Provider)}
}

// Register adds a provider. Later registrations with the same name replace the
// earlier one but keep the original position.
func (r *Registry) Register(p Provider) {
	if _, ok := r.byName[p.Name()]; ok {
		r.byName[p.Name()] = p
		for i, existing := range r.providers {
			if existing.Name() == p.Name() {
				r.providers[i] = p
				break
			}
		}
		return
	}
	r.byName[p.Name()] = p
	r.providers = append(r.providers, p)
}

// Providers returns all providers in registration order.
func (r *Registry) Providers() []Provider {
	out := make([]Provider, len(r.providers))
	copy(out, r.providers)
	return out
}

// Enabled returns the providers whose names are in names, in registration order.
// An empty names slice returns all providers. Unknown names are skipped.
func (r *Registry) Enabled(names []string) []Provider {
	if len(names) == 0 {
		return r.Providers()
	}
	want := make(map[string]bool, len(names))
	for _, n := range names {
		want[n] = true
	}
	out := make([]Provider, 0, len(names))
	for _, p := range r.providers {
		if want[p.Name()] {
			out = append(out, p)
		}
	}
	return out
}

// Names returns the registered provider names in order.
func (r *Registry) Names() []string {
	out := make([]string, len(r.providers))
	for i, p := range r.providers {
		out[i] = p.Name()
	}
	return out
}
