package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPage_HTTPContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType PageContentType
		want        string
	}{
		{
			name:        "text plain returns text/plain",
			contentType: PageContentTypeTextPlain,
			want:        "text/plain",
		},
		{
			name:        "xml returns application/xml",
			contentType: PageContentTypeXML,
			want:        "application/xml",
		},
		{
			name:        "unknown content type returns text/plain by default",
			contentType: PageContentType("UNKNOWN"),
			want:        "text/plain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Page{ContentType: tt.contentType}
			got := p.HTTPContentType()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPageList_HasMore(t *testing.T) {
	tests := []struct {
		name   string
		pl     PageList
		want   bool
	}{
		{
			name: "has more items",
			pl: PageList{
				Items:  make([]Page, 10),
				Total:  25,
				Offset: 0,
			},
			want: true,
		},
		{
			name: "has more items with offset",
			pl: PageList{
				Items:  make([]Page, 10),
				Total:  25,
				Offset: 10,
			},
			want: true,
		},
		{
			name: "exact last page",
			pl: PageList{
				Items:  make([]Page, 5),
				Total:  25,
				Offset: 20,
			},
			want: false,
		},
		{
			name: "no more items",
			pl: PageList{
				Items:  make([]Page, 10),
				Total:  10,
				Offset: 0,
			},
			want: false,
		},
		{
			name: "empty list with total zero",
			pl: PageList{
				Items:  []Page{},
				Total:  0,
				Offset: 0,
			},
			want: false,
		},
		{
			name: "empty list with total greater than zero",
			pl: PageList{
				Items:  []Page{},
				Total:  10,
				Offset: 0,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pl.HasMore()
			assert.Equal(t, tt.want, got)
		})
	}
}