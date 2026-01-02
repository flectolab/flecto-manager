package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestToken_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt *time.Time
		want      bool
	}{
		{
			name:      "nil expiration returns false",
			expiresAt: nil,
			want:      false,
		},
		{
			name:      "future expiration returns false",
			expiresAt: func() *time.Time { t := time.Now().Add(time.Hour); return &t }(),
			want:      false,
		},
		{
			name:      "past expiration returns true",
			expiresAt: func() *time.Time { t := time.Now().Add(-time.Hour); return &t }(),
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &Token{
				ExpiresAt: tt.expiresAt,
			}
			assert.Equal(t, tt.want, token.IsExpired())
		})
	}
}

func TestToken_GetRoleCode(t *testing.T) {
	tests := []struct {
		name      string
		tokenName string
		want      string
	}{
		{
			name:      "simple name",
			tokenName: "mytoken",
			want:      "token_mytoken",
		},
		{
			name:      "name with underscore",
			tokenName: "my_token",
			want:      "token_my_token",
		},
		{
			name:      "name with dash",
			tokenName: "my-token",
			want:      "token_my-token",
		},
		{
			name:      "empty name",
			tokenName: "",
			want:      "token_",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &Token{
				Name: tt.tokenName,
			}
			assert.Equal(t, tt.want, token.GetRoleCode())
		})
	}
}

func TestTokenConstants(t *testing.T) {
	assert.Equal(t, "flecto_", TokenPrefix)
	assert.Equal(t, 300, TokenNameMaxLength)
}

func TestGenerateTokenPreview(t *testing.T) {
	tests := []struct {
		name       string
		plainToken string
		want       string
	}{
		{
			name:       "normal token generates preview",
			plainToken: "flecto_abcdefghijklmnopqrstuvwxyz",
			want:       "flecto_abcd...wxyz",
		},
		{
			name:       "short token returns as-is",
			plainToken: "flecto_ab",
			want:       "flecto_ab",
		},
		{
			name:       "exactly at threshold returns as-is",
			plainToken: "flecto_abcdwxyz",
			want:       "flecto_abcdwxyz",
		},
		{
			name:       "empty string returns empty",
			plainToken: "",
			want:       "",
		},
		{
			name:       "token just over threshold",
			plainToken: "flecto_abcdewxyz",
			want:       "flecto_abcd...wxyz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateTokenPreview(tt.plainToken)
			assert.Equal(t, tt.want, got)
		})
	}
}
