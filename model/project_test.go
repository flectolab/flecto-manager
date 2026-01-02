package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProjectConstants(t *testing.T) {
	assert.Equal(t, "project_code", ColumnProjectCode)
}
