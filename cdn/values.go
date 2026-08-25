package cdn

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
)

// Values is a provider configuration, keyed by field name. It is deliberately
// untyped: the fields a provider needs are described at runtime by its
// Descriptor, not by a Go struct, which is what lets the admin UI render a form
// for a provider the front-end knows nothing about.
//
// Values come back from a JSON column, so numbers arrive as float64 and lists as
// []any. The accessors below absorb that rather than making every caller do it.
type Values map[string]any

func (v Values) Has(name string) bool {
	_, ok := v[name]
	return ok
}

func (v Values) String(name string) string {
	switch value := v[name].(type) {
	case string:
		return value
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", value)
	}
}

func (v Values) Bool(name string) bool {
	switch value := v[name].(type) {
	case bool:
		return value
	case string:
		parsed, err := strconv.ParseBool(value)
		return err == nil && parsed
	default:
		return false
	}
}

func (v Values) Int(name string) int {
	switch value := v[name].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	case string:
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return 0
		}
		return parsed
	default:
		return 0
	}
}

func (v Values) StringList(name string) []string {
	switch value := v[name].(type) {
	case []string:
		return value
	case []any:
		list := make([]string, 0, len(value))
		for _, item := range value {
			if s, ok := item.(string); ok {
				list = append(list, s)
			}
		}
		return list
	default:
		return nil
	}
}

// Clone returns a shallow copy, so that merging or redacting a configuration
// never mutates the one held by the caller.
func (v Values) Clone() Values {
	clone := make(Values, len(v))
	for name, value := range v {
		clone[name] = value
	}
	return clone
}

// MarshalGQL writes the values as a JSON object. Only the settings of a
// configuration are ever marshalled: the secrets live in their own map and never
// reach a client.
func (v Values) MarshalGQL(w io.Writer) {
	if v == nil {
		_, _ = w.Write([]byte("{}"))
		return
	}

	encoded, err := json.Marshal(map[string]any(v))
	if err != nil {
		_, _ = w.Write([]byte("{}"))
		return
	}
	_, _ = w.Write(encoded)
}

// UnmarshalGQL reads a submitted configuration. Values arrive as a JSON object,
// which is the only shape a provider configuration can take.
func (v *Values) UnmarshalGQL(raw any) error {
	object, ok := raw.(map[string]any)
	if !ok {
		return fmt.Errorf("cdn values must be an object, got %T", raw)
	}

	*v = Values(object)
	return nil
}
