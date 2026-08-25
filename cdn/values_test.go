package cdn

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValues_Has(t *testing.T) {
	values := Values{"zone_id": "abc", "empty": nil}

	assert.True(t, values.Has("zone_id"))
	assert.True(t, values.Has("empty"))
	assert.False(t, values.Has("missing"))
}

func TestValues_String(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{name: "successWithString", value: "abc", want: "abc"},
		{name: "successWithNil", value: nil, want: ""},
		{name: "successWithNumber", value: 12, want: "12"},
		{name: "successWithBool", value: true, want: "true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Values{"field": tt.value}.String("field"))
		})
	}

	assert.Equal(t, "", Values{}.String("missing"))
}

func TestValues_Bool(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  bool
	}{
		{name: "successWithBool", value: true, want: true},
		{name: "successWithStringTrue", value: "true", want: true},
		{name: "successWithStringFalse", value: "false", want: false},
		{name: "successWithInvalidString", value: "yes please", want: false},
		{name: "successWithOtherType", value: 1, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Values{"field": tt.value}.Bool("field"))
		})
	}
}

func TestValues_Int(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  int
	}{
		{name: "successWithInt", value: 12, want: 12},
		{name: "successWithInt64", value: int64(12), want: 12},
		// A value read back from a JSON column arrives as float64.
		{name: "successWithFloat64", value: float64(12), want: 12},
		{name: "successWithString", value: "12", want: 12},
		{name: "successWithInvalidString", value: "twelve", want: 0},
		{name: "successWithOtherType", value: true, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Values{"field": tt.value}.Int("field"))
		})
	}
}

func TestValues_StringList(t *testing.T) {
	t.Run("successWithStringSlice", func(t *testing.T) {
		assert.Equal(t, []string{"a", "b"}, Values{"field": []string{"a", "b"}}.StringList("field"))
	})

	t.Run("successWithAnySlice", func(t *testing.T) {
		assert.Equal(t, []string{"a", "b"}, Values{"field": []any{"a", "b"}}.StringList("field"))
	})

	t.Run("successSkipsNonStringItems", func(t *testing.T) {
		assert.Equal(t, []string{"a"}, Values{"field": []any{"a", 1}}.StringList("field"))
	})

	t.Run("successWithOtherType", func(t *testing.T) {
		assert.Nil(t, Values{"field": "a"}.StringList("field"))
	})

	t.Run("successAfterJSONRoundTrip", func(t *testing.T) {
		encoded, err := json.Marshal(Values{"hosts": []string{"a.example.com"}})
		assert.NoError(t, err)

		var decoded Values
		assert.NoError(t, json.Unmarshal(encoded, &decoded))

		assert.Equal(t, []string{"a.example.com"}, decoded.StringList("hosts"))
	})
}

func TestValues_Clone(t *testing.T) {
	original := Values{"zone_id": "abc"}

	clone := original.Clone()
	clone["zone_id"] = "changed"

	assert.Equal(t, "abc", original.String("zone_id"))
	assert.Equal(t, "changed", clone.String("zone_id"))
}

func TestValues_MarshalGQL(t *testing.T) {
	tests := []struct {
		name   string
		values Values
		want   string
	}{
		{
			name:   "success",
			values: Values{"zone_id": "abc123"},
			want:   `{"zone_id":"abc123"}`,
		},
		{
			name:   "successWithNilValues",
			values: nil,
			want:   `{}`,
		},
		{
			name:   "successWithEmptyValues",
			values: Values{},
			want:   `{}`,
		},
		{
			name:   "successWithUnencodableValue",
			values: Values{"broken": make(chan int)},
			want:   `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}

			tt.values.MarshalGQL(out)

			assert.JSONEq(t, tt.want, out.String())
		})
	}
}

func TestValues_UnmarshalGQL(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var values Values

		err := values.UnmarshalGQL(map[string]any{"zone_id": "abc123"})

		assert.NoError(t, err)
		assert.Equal(t, "abc123", values.String("zone_id"))
	})

	t.Run("failWithNonObject", func(t *testing.T) {
		var values Values

		err := values.UnmarshalGQL("abc123")

		assert.ErrorContains(t, err, "must be an object")
	})
}
