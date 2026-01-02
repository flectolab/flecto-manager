package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPtr(t *testing.T) {
	val := "test"
	assert.Equal(t, &val, Ptr(val))
}
