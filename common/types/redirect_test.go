package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedirect_HTTPCode(t *testing.T) {
	tests := []struct {
		name   string
		status RedirectStatus
		want   int
	}{
		{
			name:   "moved permanent returns 301",
			status: RedirectStatusMovedPermanent,
			want:   301,
		},
		{
			name:   "found returns 302",
			status: RedirectStatusFound,
			want:   302,
		},
		{
			name:   "temporary redirect returns 307",
			status: RedirectStatusTemporary,
			want:   307,
		},
		{
			name:   "permanent redirect returns 308",
			status: RedirectStatusPermanent,
			want:   308,
		},
		{
			name:   "unknown status returns 302 by default",
			status: RedirectStatus("UNKNOWN"),
			want:   302,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Redirect{Status: tt.status}
			got := r.HTTPCode()
			assert.Equal(t, tt.want, got)
		})
	}
}