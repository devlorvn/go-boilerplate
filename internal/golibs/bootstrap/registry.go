package bootstrap

import (
	"context"
	"fmt"
	"sort"

	"go-boilerplate/internal/golibs/configs"
)

type runFunc func(ctx context.Context, configDir string) error

var registry = map[string]runFunc{}

// Register makes a service runnable as `server gserver <name>`. Call it from
// an init() in cmd/server/<name>; the config type is inferred from s.
func Register[T configs.Provider](name string, s Servicer[T]) {
	if _, dup := registry[name]; dup {
		panic(fmt.Sprintf("bootstrap: service %q registered twice", name))
	}
	registry[name] = func(ctx context.Context, configDir string) error {
		return run(ctx, name, configDir, s)
	}
}

func Names() []string {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// Run starts the registered service and blocks until ctx is cancelled or one
// of its servers fails.
func Run(ctx context.Context, name, configDir string) error {
	fn, ok := registry[name]
	if !ok {
		return fmt.Errorf("unknown service %q, available: %v", name, Names())
	}
	return fn(ctx, configDir)
}
