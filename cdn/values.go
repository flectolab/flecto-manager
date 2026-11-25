package cdn

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
)

// Values is a provider configuration keyed by field name, untyped since fields are
// described at runtime by the Descriptor. Read from JSON, numbers arrive as
// float64 and lists as []any: the accessors absorb that.
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

// Clone returns a shallow copy.
func (v Values) Clone() Values {
	clone := make(Values, len(v))
	for name, value := range v {
		clone[name] = value
	}
	return clone
}

// MarshalGQL writes the values as a JSON object.
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

// UnmarshalGQL reads a submitted configuration from a JSON object.
func (v *Values) UnmarshalGQL(raw any) error {
	object, ok := raw.(map[string]any)
	if !ok {
		return fmt.Errorf("cdn values must be an object, got %T", raw)
	}

	*v = Values(object)
	return nil
}
