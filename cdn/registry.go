package cdn

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// Options are instance-wide settings, as opposed to one configuration's values.
type Options struct {
	// HTTPTimeout bounds one API call. Zero keeps the provider's default.
	HTTPTimeout time.Duration
}

// Factory builds a provider from a validated configuration.
type Factory func(settings, secrets Values, options Options) (Provider, error)

type registration struct {
	descriptor Descriptor
	factory    Factory
}

var (
	registryMutex sync.RWMutex
	registry      = map[string]registration{}
)

// Register adds a provider type, from the init of each provider package.
func Register(descriptor Descriptor, factory Factory) {
	registryMutex.Lock()
	defer registryMutex.Unlock()

	registry[descriptor.Type] = registration{descriptor: descriptor, factory: factory}
}

// Descriptors lists the registered provider types, sorted by type.
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
