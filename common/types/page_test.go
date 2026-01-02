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