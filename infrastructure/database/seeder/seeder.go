package seeder

import "context"

// Seeder defines a named database seeding operation.
type Seeder interface {
	Name() string
	Seed(ctx context.Context) error
}

// Registry holds all seeders and supports running them by name or all at once.
type Registry struct {
	seeders map[string]Seeder
	order   []string
}

// NewRegistry creates a Registry from the given seeders. Seeders are run in
// the order they are passed.
func NewRegistry(seeders ...Seeder) *Registry {
	r := &Registry{
		seeders: make(map[string]Seeder, len(seeders)),
		order:   make([]string, 0, len(seeders)),
	}
	for _, s := range seeders {
		r.seeders[s.Name()] = s
		r.order = append(r.order, s.Name())
	}
	return r
}

// Run executes the named seeders. If no names are provided, runs all seeders
// in registration order. If names are provided, runs only those seeders in the
// order specified. Returns an error from the first failing seeder.
func (r *Registry) Run(ctx context.Context, names ...string) error {
	if len(names) == 0 {
		for _, n := range r.order {
			if err := r.seeders[n].Seed(ctx); err != nil {
				return err
			}
		}
		return nil
	}

	for _, name := range names {
		s, ok := r.seeders[name]
		if !ok {
			return &SeederNotFoundError{Name: name}
		}
		if err := s.Seed(ctx); err != nil {
			return err
		}
	}
	return nil
}

// SeederNotFoundError is returned when a requested seeder name is not registered.
type SeederNotFoundError struct {
	Name string
}

func (e *SeederNotFoundError) Error() string {
	return "seeder not found: " + e.Name
}
