package types

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Duration
		expected Duration
	}{
		{
			name:     "100 milliseconds",
			input:    100 * time.Millisecond,
			expected: Duration(100 * time.Millisecond),
		},
		{
			name:     "1 second",
			input:    time.Second,
			expected: Duration(time.Second),
		},
		{
			name:     "zero",
			input:    0,
			expected: Duration(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewDuration(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDuration_Duration(t *testing.T) {
	d := NewDuration(100 * time.Millisecond)
	assert.Equal(t, 100*time.Millisecond, d.Duration())
}

func TestDuration_String(t *testing.T) {
	tests := []struct {
		name     string
		duration Duration
		expected string
	}{
		{
			name:     "100 milliseconds",
			duration: NewDuration(100 * time.Millisecond),
			expected: "100ms",
		},
		{
			name:     "1 second",
			duration: NewDuration(time.Second),
			expected: "1s",
		},
		{
			name:     "1 minute 30 seconds",
			duration: NewDuration(90 * time.Second),
			expected: "1m30s",
		},
		{
			name:     "zero",
			duration: NewDuration(0),
			expected: "0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.duration.String())
		})
	}
}

func TestDuration_Nanoseconds(t *testing.T) {
	d := NewDuration(100 * time.Millisecond)
	assert.Equal(t, int64(100000000), d.Nanoseconds())
}

func TestDuration_Milliseconds(t *testing.T) {
	d := NewDuration(100 * time.Millisecond)
	assert.Equal(t, int64(100), d.Milliseconds())
}

func TestDuration_Seconds(t *testing.T) {
	d := NewDuration(1500 * time.Millisecond)
	assert.Equal(t, 1.5, d.Seconds())
}

func TestDuration_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		duration Duration
		expected string
	}{
		{
			name:     "100 milliseconds",
			duration: NewDuration(100 * time.Millisecond),
			expected: "100000000",
		},
		{
			name:     "1 second",
			duration: NewDuration(time.Second),
			expected: "1000000000",
		},
		{
			name:     "zero",
			duration: NewDuration(0),
			expected: "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := json.Marshal(tt.duration)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

func TestDuration_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Duration
		wantErr  bool
	}{
		{
			name:     "string format milliseconds",
			input:    `"100ms"`,
			expected: NewDuration(100 * time.Millisecond),
			wantErr:  false,
		},
		{
			name:     "string format seconds",
			input:    `"1s"`,
			expected: NewDuration(time.Second),
			wantErr:  false,
		},
		{
			name:     "string format minutes",
			input:    `"5m"`,
			expected: NewDuration(5 * time.Minute),
			wantErr:  false,
		},
		{
			name:     "string format combined",
			input:    `"1m30s"`,
			expected: NewDuration(90 * time.Second),
			wantErr:  false,
		},
		{
			name:     "number format nanoseconds",
			input:    `100000000`,
			expected: NewDuration(100 * time.Millisecond),
			wantErr:  false,
		},
		{
			name:     "number format zero",
			input:    `0`,
			expected: NewDuration(0),
			wantErr:  false,
		},
		{
			name:     "invalid string format",
			input:    `"invalid"`,
			expected: Duration(0),
			wantErr:  true,
		},
		{
			name:     "invalid json",
			input:    `[1,2,3]`,
			expected: Duration(0),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d Duration
			err := json.Unmarshal([]byte(tt.input), &d)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, d)
			}
		})
	}
}

func TestDuration_UnmarshalJSON_InStruct(t *testing.T) {
	type TestStruct struct {
		Name     string   `json:"name"`
		Duration Duration `json:"duration"`
	}

	tests := []struct {
		name     string
		input    string
		expected TestStruct
		wantErr  bool
	}{
		{
			name:  "string duration in struct",
			input: `{"name":"test","duration":"100ms"}`,
			expected: TestStruct{
				Name:     "test",
				Duration: NewDuration(100 * time.Millisecond),
			},
			wantErr: false,
		},
		{
			name:  "number duration in struct",
			input: `{"name":"test","duration":100000000}`,
			expected: TestStruct{
				Name:     "test",
				Duration: NewDuration(100 * time.Millisecond),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s TestStruct
			err := json.Unmarshal([]byte(tt.input), &s)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, s)
			}
		})
	}
}

func TestDuration_Scan(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected Duration
		wantErr  bool
	}{
		{
			name:     "int64 value",
			input:    int64(100000000),
			expected: NewDuration(100 * time.Millisecond),
			wantErr:  false,
		},
		{
			name:     "float64 value",
			input:    float64(100000000),
			expected: NewDuration(100 * time.Millisecond),
			wantErr:  false,
		},
		{
			name:     "nil value",
			input:    nil,
			expected: Duration(0),
			wantErr:  false,
		},
		{
			name:     "invalid type string",
			input:    "100ms",
			expected: Duration(0),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d Duration
			err := d.Scan(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, d)
			}
		})
	}
}
