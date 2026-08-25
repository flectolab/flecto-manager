package cdn

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTemporaryError_Error(t *testing.T) {
	t.Run("successWithoutDelay", func(t *testing.T) {
		err := Temporary(errors.New("boom"), 0)

		assert.Equal(t, "boom", err.Error())
	})

	t.Run("successWithDelay", func(t *testing.T) {
		err := Temporary(errors.New("boom"), 30*time.Second)

		assert.Equal(t, "boom (retry after 30s)", err.Error())
	})
}

func TestTemporaryError_Unwrap(t *testing.T) {
	sentinel := errors.New("boom")

	err := Temporary(sentinel, 0)

	assert.ErrorIs(t, err, sentinel)
}

func TestAsTemporary(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		wantTemporary  bool
		wantRetryAfter time.Duration
	}{
		{
			name:           "successWithTemporaryError",
			err:            Temporary(errors.New("boom"), 30*time.Second),
			wantTemporary:  true,
			wantRetryAfter: 30 * time.Second,
		},
		{
			name:          "successWithWrappedTemporaryError",
			err:           fmt.Errorf("purge failed: %w", Temporary(errors.New("boom"), 0)),
			wantTemporary: true,
		},
		{
			name: "failWithPermanentError",
			err:  errors.New("boom"),
		},
		{
			name: "failWithNilError",
			err:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retryAfter, temporary := AsTemporary(tt.err)

			assert.Equal(t, tt.wantTemporary, temporary)
			assert.Equal(t, tt.wantRetryAfter, retryAfter)
		})
	}
}
