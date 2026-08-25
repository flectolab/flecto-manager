package cdn

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// Options are the settings of the instance a provider is built with, as opposed to
// the values of one configuration.
type Options struct {
	// HTTPTimeout bounds a single call to the provider's API. Zero leaves the
	// provider's own default.
	HTTPTimeout time.Duration
}

// Factory builds a provider from a validated configuration. Settings and secrets
// arrive separated the way they are stored, so a factory never has to know which
// of its fields were encrypted.
type Factory func(settings, secrets Values, options Options) (Provider, error)

type registration struct {
	descriptor Descriptor
	factory    Factory
}

var (
	registryMutex sync.RWMutex
	registry      = map[string]registration{}
)

// Register adds a provider type. It is called from the init of each provider
// package, which the application imports for its side effect - the same shape as
// the database dialector registry.
func Register(descriptor Descriptor, factory Factory) {
	registryMutex.Lock()
	defer registryMutex.Unlock()

	registry[descriptor.Type] = registration{descriptor: descriptor, factory: factory}
}

// Descriptors lists the registered provider types, sorted by type so that the
// admin UI always offers them in the same order.
func Descriptors() []Descriptor {
	registryMutex.RLock()
	defer registryMutex.RUnlock()

	descriptors := make([]Descriptor, 0, len(registry))
	for _, entry := range registry {
		descriptors = append(descriptors, entry.descriptor)
	}
	sort.Slice(descriptors, func(i, j int) bool {
		return descriptors[i].Type < descriptors[j].Type
	})

	return descriptors
}

// GetDescriptor returns the descriptor of a provider type.
func GetDescriptor(providerType string) (Descriptor, error) {
	registryMutex.RLock()
	defer registryMutex.RUnlock()

	entry, ok := registry[providerType]
	if !ok {
		return Descriptor{}, fmt.Errorf("unknown cdn provider type %q", providerType)
	}
	return entry.descriptor, nil
}

// New validates a configuration against its descriptor and builds the provider.
// Validation happens here rather than in each factory so that every provider gets
// the same guarantees, including the ones added later.
func New(providerType string, settings, secrets Values, options Options) (Provider, error) {
	registryMutex.RLock()
	entry, ok := registry[providerType]
	registryMutex.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown cdn provider type %q", providerType)
	}

	values := entry.descriptor.ApplyDefaults(merge(settings, secrets))
	if err := entry.descriptor.Validate(values); err != nil {
		return nil, err
	}

	filledSettings, filledSecrets := entry.descriptor.Split(values)
	return entry.factory(filledSettings, filledSecrets, options)
}

func merge(settings, secrets Values) Values {
	merged := settings.Clone()
	for name, value := range secrets {
		merged[name] = value
	}
	return merged
}
