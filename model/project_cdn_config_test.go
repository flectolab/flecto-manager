package model

import (
	"testing"

	"github.com/flectolab/flecto-manager/cdn"
	"github.com/flectolab/flecto-manager/types"
	"github.com/stretchr/testify/assert"
)

func TestProjectCdnConfig_IsEnabled(t *testing.T) {
	tests := []struct {
		name   string
		config *ProjectCdnConfig
		want   bool
	}{
		{name: "successWhenEnabled", config: &ProjectCdnConfig{Enabled: types.Ptr(true)}, want: true},
		{name: "successWhenDisabled", config: &ProjectCdnConfig{Enabled: types.Ptr(false)}, want: false},
		// A row written before the column existed is not a reason to stop purging.
		{name: "successWhenUnset", config: &ProjectCdnConfig{}, want: true},
		{name: "successWithNilConfig", config: nil, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.config.IsEnabled())
		})
	}
}

func TestProjectCdnConfig_Hosts(t *testing.T) {
	tests := []struct {
		name   string
		config *ProjectCdnConfig
		want   []string
	}{
		{
			name:   "success",
			config: &ProjectCdnConfig{Settings: cdn.Values{cdn.FieldHosts: []string{"www.example.com"}}},
			want:   []string{"www.example.com"},
		},
		{
			name:   "successWithoutHosts",
			config: &ProjectCdnConfig{Settings: cdn.Values{}},
			want:   nil,
		},
		{
			name:   "successWithNilConfig",
			config: nil,
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.config.Hosts())
		})
	}
}
