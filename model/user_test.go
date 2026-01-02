package model

import (
	"testing"

	"github.com/flectolab/flecto-manager/types"
	"github.com/stretchr/testify/assert"
)

func TestUser_IsActive(t *testing.T) {
	tests := []struct {
		name   string
		active *bool
		want   bool
	}{
		{
			name:   "nil active returns false",
			active: nil,
			want:   false,
		},
		{
			name:   "true active returns true",
			active: types.Ptr(true),
			want:   true,
		},
		{
			name:   "false active returns false",
			active: types.Ptr(false),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{
				Active: tt.active,
			}
			assert.Equal(t, tt.want, user.IsActive())
		})
	}
}

func TestUser_HasPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		want     bool
	}{
		{
			name:     "empty password returns false",
			password: "",
			want:     false,
		},
		{
			name:     "non-empty password returns true",
			password: "hashed_password",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{
				Password: tt.password,
			}
			assert.Equal(t, tt.want, user.HasPassword())
		})
	}
}
