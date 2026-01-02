package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResourcePermission_TableName(t *testing.T) {
	assert.Equal(t, "resource_permissions", ResourcePermission{}.TableName())
}

func TestAdminPermission_TableName(t *testing.T) {
	assert.Equal(t, "admin_permissions", AdminPermission{}.TableName())
}
