package cdn

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCapabilities_Supports(t *testing.T) {
	tests := []struct {
		name         string
		capabilities Capabilities
		kind         TargetKind
		want         bool
	}{
		{
			name:         "successWithURL",
			capabilities: Capabilities{URL: true},
			kind:         KindURL,
			want:         true,
		},
		{
			name:         "successWithPrefix",
			capabilities: Capabilities{Prefix: true},
			kind:         KindPrefix,
			want:         true,
		},
		{
			name:         "successWithHostname",
			capabilities: Capabilities{Hostname: true},
			kind:         KindHostname,
			want:         true,
		},
		{
			name:         "failWithUnsupportedKind",
			capabilities: Capabilities{URL: true},
			kind:         KindPrefix,
			want:         false,
		},
		{
			name:         "failWithUnknownKind",
			capabilities: Capabilities{URL: true, Prefix: true, Hostname: true},
			kind:         TargetKind("TAG"),
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.capabilities.Supports(tt.kind))
		})
	}
}
