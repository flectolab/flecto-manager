package cdn

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// stubProvider is the minimum a factory has to return for the registry tests.
type stubProvider struct {
	settings Values
	secrets  Values
	options  Options
}

func (p *stubProvider) Type() string                          { return "test" }
func (p *stubProvider) Capabilities() Capabilities            { return Capabilities{URL: true} }
func (p *stubProvider) Limits() Limits                        { return Limits{MaxTargetsPerRequest: 100} }
func (p *stubProvider) Verify(context.Context) error          { return nil }
func (p *stubProvider) Purge(context.Context, []Target) error { return nil }

// withRegistry installs a registry holding only what the case needs and restores
// the real one afterwards, since the registry is global.
func withRegistry(t *testing.T) {
	t.Helper()

	registryMutex.Lock()
	previous := registry
	registry = map[string]registration{}
	registryMutex.Unlock()

	t.Cleanup(func() {
		registryMutex.Lock()
		registry = previous
		registryMutex.Unlock()
	})
}

func TestRegister(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		withRegistry(t)
		Register(testDescriptor(), func(settings, secrets Values, options Options) (Provider, error) {
			return &stubProvider{}, nil
		})

		got, err := GetDescriptor("test")

		assert.NoError(t, err)
		assert.Equal(t, "test", got.Type)
	})

	t.Run("successReplacesPreviousRegistration", func(t *testing.T) {
		withRegistry(t)
		Register(testDescriptor(), func(settings, secrets Values, options Options) (Provider, error) {
			return &stubProvider{}, nil
		})

		updated := testDescriptor()
		updated.Label = "Updated"
		Register(updated, func(settings, secrets Values, options Options) (Provider, error) {
			return &stubProvider{}, nil
		})

		got, err := GetDescriptor("test")

		assert.NoError(t, err)
		assert.Equal(t, "Updated", got.Label)
	})
}

func TestDescriptors(t *testing.T) {
	t.Run("successSortsByType", func(t *testing.T) {
		withRegistry(t)
		factory := func(settings, secrets Values, options Options) (Provider, error) { return &stubProvider{}, nil }

		second := testDescriptor()
		second.Type = "zzz"
		Register(second, factory)
		Register(testDescriptor(), factory)

		got := Descriptors()

		assert.Len(t, got, 2)
		assert.Equal(t, "test", got[0].Type)
		assert.Equal(t, "zzz", got[1].Type)
	})

	t.Run("successWithEmptyRegistry", func(t *testing.T) {
		withRegistry(t)

		assert.Empty(t, Descriptors())
	})
}

func TestGetDescriptor(t *testing.T) {
	t.Run("failWithUnknownType", func(t *testing.T) {
		withRegistry(t)

		_, err := GetDescriptor("nope")

		assert.ErrorContains(t, err, `unknown cdn provider type "nope"`)
	})
}

func TestNew(t *testing.T) {
	factory := func(settings, secrets Values, options Options) (Provider, error) {
		return &stubProvider{settings: settings, secrets: secrets, options: options}, nil
	}

	t.Run("successPassesSplitValuesToFactory", func(t *testing.T) {
		withRegistry(t)
		Register(testDescriptor(), factory)

		got, err := New("test", Values{"zone_id": "abc123", "hosts": []string{"a.com"}}, Values{"api_token": "s3cret"}, Options{})

		assert.NoError(t, err)
		provider := got.(*stubProvider)
		assert.Equal(t, "abc123", provider.settings.String("zone_id"))
		assert.Equal(t, "s3cret", provider.secrets.String("api_token"))
	})

	t.Run("successPassesTheOptionsToFactory", func(t *testing.T) {
		withRegistry(t)
		Register(testDescriptor(), factory)

		got, err := New("test", Values{"zone_id": "abc123", "hosts": []string{"a.com"}}, Values{"api_token": "s3cret"},
			Options{HTTPTimeout: 3 * time.Second})

		assert.NoError(t, err)
		assert.Equal(t, 3*time.Second, got.(*stubProvider).options.HTTPTimeout)
	})

	t.Run("successAppliesDefaults", func(t *testing.T) {
		withRegistry(t)
		Register(testDescriptor(), factory)

		got, err := New("test", Values{"zone_id": "abc123", "hosts": []string{"a.com"}}, Values{"api_token": "s3cret"}, Options{})

		assert.NoError(t, err)
		assert.Equal(t, 10, got.(*stubProvider).settings.Int("timeout"))
	})

	t.Run("failWithUnknownType", func(t *testing.T) {
		withRegistry(t)

		_, err := New("nope", Values{}, Values{}, Options{})

		assert.ErrorContains(t, err, `unknown cdn provider type "nope"`)
	})

	t.Run("failWithInvalidConfiguration", func(t *testing.T) {
		withRegistry(t)
		Register(testDescriptor(), factory)

		_, err := New("test", Values{"hosts": []string{"a.com"}}, Values{"api_token": "s3cret"}, Options{})

		assert.ErrorContains(t, err, "zone_id is required")
	})

	t.Run("failWhenFactoryFails", func(t *testing.T) {
		withRegistry(t)
		Register(testDescriptor(), func(settings, secrets Values, options Options) (Provider, error) {
			return nil, errors.New("boom")
		})

		_, err := New("test", Values{"zone_id": "abc", "hosts": []string{"a.com"}}, Values{"api_token": "s3cret"}, Options{})

		assert.ErrorContains(t, err, "boom")
	})
}
