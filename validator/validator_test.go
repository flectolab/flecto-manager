package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	validator := New()
	assert.NotNil(t, validator)
}
