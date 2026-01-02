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

func TestRedirectList_HasMore(t *testing.T) {
	tests := []struct {
		name string
		rl   RedirectList
		want bool
	}{
		{
			name: "has more items",
			rl: RedirectList{
				Items:  make([]Redirect, 10),
				Total:  25,
				Offset: 0,
			},
			want: true,
		},
		{
			name: "has more items with offset",
			rl: RedirectList{
				Items:  make([]Redirect, 10),
				Total:  25,
				Offset: 10,
			},
			want: true,
		},
		{
			name: "exact last page",
			rl: RedirectList{
				Items:  make([]Redirect, 5),
				Total:  25,
				Offset: 20,
			},
			want: false,
		},
		{
			name: "no more items",
			rl: RedirectList{
				Items:  make([]Redirect, 10),
				Total:  10,
				Offset: 0,
			},
			want: false,
		},
		{
			name: "empty list with total zero",
			rl: RedirectList{
				Items:  []Redirect{},
				Total:  0,
				Offset: 0,
			},
			want: false,
		},
		{
			name: "empty list with total greater than zero",
			rl: RedirectList{
				Items:  []Redirect{},
				Total:  10,
				Offset: 0,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.rl.HasMore()
			assert.Equal(t, tt.want, got)
		})
	}
}