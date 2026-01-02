package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPageTreeMatcher(t *testing.T) {
	tree := NewPageTreeMatcher()

	assert.NotNil(t, tree)

	pt, ok := tree.(*PageTree)
	assert.True(t, ok, "NewPageTreeMatcher() should return *PageTree")

	assert.NotNil(t, pt.basicHost)
	assert.NotNil(t, pt.basic)
}

func TestPageTree_Insert(t *testing.T) {
	tests := []struct {
		name  string
		pages []*Page
	}{
		{
			name: "insert basic page",
			pages: []*Page{
				{Type: PageTypeBasic, Path: "/robots.txt", Content: "User-agent: *", ContentType: PageContentTypeTextPlain},
			},
		},
		{
			name: "insert basic host page",
			pages: []*Page{
				{Type: PageTypeBasicHost, Path: "example.com/robots.txt", Content: "User-agent: *", ContentType: PageContentTypeTextPlain},
			},
		},
		{
			name: "insert multiple pages",
			pages: []*Page{
				{Type: PageTypeBasic, Path: "/robots.txt", Content: "User-agent: *", ContentType: PageContentTypeTextPlain},
				{Type: PageTypeBasic, Path: "/sitemap.xml", Content: "<sitemap/>", ContentType: PageContentTypeXML},
				{Type: PageTypeBasicHost, Path: "example.com/robots.txt", Content: "User-agent: Googlebot", ContentType: PageContentTypeTextPlain},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := NewPageTreeMatcher()

			for _, p := range tt.pages {
				tree.Insert(p)
			}
		})
	}
}

func TestPageTree_Match(t *testing.T) {
	tests := []struct {
		name        string
		pages       []*Page
		host        string
		uri         string
		wantPage    bool
		wantContent string
	}{
		{
			name: "match basic page",
			pages: []*Page{
				{Type: PageTypeBasic, Path: "/robots.txt", Content: "User-agent: *", ContentType: PageContentTypeTextPlain},
			},
			host:        "example.com",
			uri:         "/robots.txt",
			wantPage:    true,
			wantContent: "User-agent: *",
		},
		{
			name: "match basic host page",
			pages: []*Page{
				{Type: PageTypeBasicHost, Path: "example.com/robots.txt", Content: "User-agent: Googlebot", ContentType: PageContentTypeTextPlain},
			},
			host:        "example.com",
			uri:         "/robots.txt",
			wantPage:    true,
			wantContent: "User-agent: Googlebot",
		},
		{
			name: "no match returns nil",
			pages: []*Page{
				{Type: PageTypeBasic, Path: "/robots.txt", Content: "User-agent: *", ContentType: PageContentTypeTextPlain},
			},
			host:        "example.com",
			uri:         "/sitemap.xml",
			wantPage:    false,
			wantContent: "",
		},
		{
			name: "basic host has priority over basic",
			pages: []*Page{
				{Type: PageTypeBasic, Path: "/robots.txt", Content: "basic content", ContentType: PageContentTypeTextPlain},
				{Type: PageTypeBasicHost, Path: "example.com/robots.txt", Content: "host content", ContentType: PageContentTypeTextPlain},
			},
			host:        "example.com",
			uri:         "/robots.txt",
			wantPage:    true,
			wantContent: "host content",
		},
		{
			name: "basic page matches when host does not match",
			pages: []*Page{
				{Type: PageTypeBasic, Path: "/robots.txt", Content: "basic content", ContentType: PageContentTypeTextPlain},
				{Type: PageTypeBasicHost, Path: "other.com/robots.txt", Content: "host content", ContentType: PageContentTypeTextPlain},
			},
			host:        "example.com",
			uri:         "/robots.txt",
			wantPage:    true,
			wantContent: "basic content",
		},
		{
			name: "match sitemap.xml",
			pages: []*Page{
				{Type: PageTypeBasic, Path: "/sitemap.xml", Content: "<?xml version=\"1.0\"?><urlset/>", ContentType: PageContentTypeXML},
			},
			host:        "example.com",
			uri:         "/sitemap.xml",
			wantPage:    true,
			wantContent: "<?xml version=\"1.0\"?><urlset/>",
		},
		{
			name:        "empty tree returns nil",
			pages:       []*Page{},
			host:        "example.com",
			uri:         "/robots.txt",
			wantPage:    false,
			wantContent: "",
		},
		{
			name: "match with empty uri",
			pages: []*Page{
				{Type: PageTypeBasic, Path: "", Content: "root content", ContentType: PageContentTypeTextPlain},
			},
			host:        "example.com",
			uri:         "",
			wantPage:    true,
			wantContent: "root content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := NewPageTreeMatcher()

			for _, p := range tt.pages {
				tree.Insert(p)
			}

			got := tree.Match(tt.host, tt.uri)

			if tt.wantPage {
				assert.NotNil(t, got)
				assert.Equal(t, tt.wantContent, got.Content)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}