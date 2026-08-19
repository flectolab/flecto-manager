package model

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestActivityData_Value(t *testing.T) {
	tests := []struct {
		name string
		data ActivityData
		want any
	}{
		{name: "payload", data: ActivityData(`{"a":1}`), want: `{"a":1}`},
		{name: "empty", data: ActivityData(``), want: nil},
		{name: "nil", data: nil, want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := tt.data.Value()
			assert.NoError(t, err)
			assert.Equal(t, tt.want, value)
		})
	}
}

func TestActivityData_Scan(t *testing.T) {
	t.Run("from bytes", func(t *testing.T) {
		var data ActivityData
		assert.NoError(t, data.Scan([]byte(`{"a":1}`)))
		assert.Equal(t, `{"a":1}`, string(data))
	})

	t.Run("scanned bytes are copied", func(t *testing.T) {
		source := []byte(`{"a":1}`)
		var data ActivityData
		assert.NoError(t, data.Scan(source))

		// The driver may reuse its buffer between rows.
		copy(source, []byte(`{"b":2}`))
		assert.Equal(t, `{"a":1}`, string(data))
	})

	t.Run("from string", func(t *testing.T) {
		var data ActivityData
		assert.NoError(t, data.Scan(`{"a":1}`))
		assert.Equal(t, `{"a":1}`, string(data))
	})

	t.Run("from nil", func(t *testing.T) {
		data := ActivityData(`{"a":1}`)
		assert.NoError(t, data.Scan(nil))
		assert.Nil(t, data)
	})

	t.Run("unsupported type", func(t *testing.T) {
		var data ActivityData
		assert.ErrorContains(t, data.Scan(42), "cannot scan int into ActivityData")
	})
}

func TestActivityData_JSON(t *testing.T) {
	t.Run("marshals as raw json, not base64", func(t *testing.T) {
		out, err := json.Marshal(struct {
			Data ActivityData `json:"data"`
		}{Data: ActivityData(`{"a":1}`)})
		assert.NoError(t, err)
		assert.JSONEq(t, `{"data":{"a":1}}`, string(out))
	})

	t.Run("empty marshals to null", func(t *testing.T) {
		out, err := ActivityData(nil).MarshalJSON()
		assert.NoError(t, err)
		assert.Equal(t, "null", string(out))
	})

	t.Run("round trip", func(t *testing.T) {
		var data ActivityData
		assert.NoError(t, json.Unmarshal([]byte(`{"a":1}`), &data))
		assert.JSONEq(t, `{"a":1}`, string(data))
	})
}

func TestActivityData_GQL(t *testing.T) {
	t.Run("marshals the payload as-is", func(t *testing.T) {
		buf := &bytes.Buffer{}
		ActivityData(`{"a":1}`).MarshalGQL(buf)
		assert.Equal(t, `{"a":1}`, buf.String())
	})

	t.Run("empty marshals to null", func(t *testing.T) {
		buf := &bytes.Buffer{}
		ActivityData(nil).MarshalGQL(buf)
		assert.Equal(t, "null", buf.String())
	})

	t.Run("unmarshal is refused", func(t *testing.T) {
		var data ActivityData
		assert.ErrorIs(t, data.UnmarshalGQL(`{"a":1}`), ErrActivityDataReadOnly)
	})
}
