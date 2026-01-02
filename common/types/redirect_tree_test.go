package types

import (
	"regexp/syntax"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRedirectTreeMatcher(t *testing.T) {
	tree := NewRedirectTreeMatcher()

	assert.NotNil(t, tree)

	rt, ok := tree.(*RedirectTree)
	assert.True(t, ok, "NewRedirectTreeMatcher() should return *RedirectTree")

	assert.NotNil(t, rt.basicHost)
	assert.NotNil(t, rt.basic)
	assert.NotNil(t, rt.regexHost)
	assert.NotNil(t, rt.regex)
	assert.NotNil(t, rt.regexHostRoot)
	assert.NotNil(t, rt.regexRoot)
}

func TestRedirectTree_Insert(t *testing.T) {
	tests := []struct {
		name      string
		redirects []*Redirect
		wantErr   bool
	}{
		{
			name: "insert basic redirect",
			redirects: []*Redirect{
				{Type: RedirectTypeBasic, Source: "/old", Target: "/new", Status: RedirectStatusMovedPermanent},
			},
			wantErr: false,
		},
		{
			name: "insert basic host redirect",
			redirects: []*Redirect{
				{Type: RedirectTypeBasicHost, Source: "example.com/old", Target: "/new", Status: RedirectStatusMovedPermanent},
			},
			wantErr: false,
		},
		{
			name: "insert regex redirect with prefix",
			redirects: []*Redirect{
				{Type: RedirectTypeRegex, Source: "/product/[0-9]+", Target: "/item/$1", Status: RedirectStatusMovedPermanent},
			},
			wantErr: false,
		},
		{
			name: "insert regex redirect without prefix",
			redirects: []*Redirect{
				{Type: RedirectTypeRegex, Source: "[a-z]+/item", Target: "/new", Status: RedirectStatusMovedPermanent},
			},
			wantErr: false,
		},
		{
			name: "insert regex host redirect",
			redirects: []*Redirect{
				{Type: RedirectTypeRegexHost, Source: "example.com/user/[0-9]+", Target: "/profile/$1", Status: RedirectStatusMovedPermanent},
			},
			wantErr: false,
		},
		{
			name: "insert regex host redirect without prefix",
			redirects: []*Redirect{
				{Type: RedirectTypeRegexHost, Source: "[a-z]+.com/path", Target: "/new", Status: RedirectStatusMovedPermanent},
			},
			wantErr: false,
		},
		{
			name: "insert invalid regex returns error",
			redirects: []*Redirect{
				{Type: RedirectTypeRegex, Source: "[invalid(regex", Target: "/new", Status: RedirectStatusMovedPermanent},
			},
			wantErr: true,
		},
		{
			name: "insert multiple redirects with same prefix",
			redirects: []*Redirect{
				{Type: RedirectTypeRegex, Source: "/product/[0-9]+", Target: "/item1", Status: RedirectStatusMovedPermanent},
				{Type: RedirectTypeRegex, Source: "/product/[a-z]+", Target: "/item2", Status: RedirectStatusMovedPermanent},
			},
			wantErr: false,
		},
		{
			name: "insert all types",
			redirects: []*Redirect{
				{Type: RedirectTypeBasic, Source: "/basic", Target: "/new-basic", Status: RedirectStatusMovedPermanent},
				{Type: RedirectTypeBasicHost, Source: "host.com/path", Target: "/new-host", Status: RedirectStatusFound},
				{Type: RedirectTypeRegex, Source: "/regex/[0-9]+", Target: "/new-regex", Status: RedirectStatusTemporary},
				{Type: RedirectTypeRegexHost, Source: "host.com/regex/[0-9]+", Target: "/new-regex-host", Status: RedirectStatusPermanent},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := NewRedirectTreeMatcher()

			for _, r := range tt.redirects {
				err := tree.Insert(r)
				if tt.wantErr {
					assert.Error(t, err)
					return
				}
				assert.NoError(t, err)
			}
		})
	}
}

func TestRedirectTree_Match(t *testing.T) {
	tests := []struct {
		name         string
		redirects    []*Redirect
		host         string
		uri          string
		wantRedirect bool
		wantTarget   string
	}{
		{
			name: "match basic redirect",
			redirects: []*Redirect{
				{Type: RedirectTypeBasic, Source: "/old-page", Target: "/new-page", Status: RedirectStatusMovedPermanent},
			},
			host:         "example.com",
			uri:          "/old-page",
			wantRedirect: true,
			wantTarget:   "/new-page",
		},
		{
			name: "match basic host redirect",
			redirects: []*Redirect{
				{Type: RedirectTypeBasicHost, Source: "example.com/old-page", Target: "/new-page", Status: RedirectStatusMovedPermanent},
			},
			host:         "example.com",
			uri:          "/old-page",
			wantRedirect: true,
			wantTarget:   "/new-page",
		},
		{
			name: "match regex redirect",
			redirects: []*Redirect{
				{Type: RedirectTypeRegex, Source: "/product/([0-9]+)", Target: "/item/$1", Status: RedirectStatusMovedPermanent},
			},
			host:         "example.com",
			uri:          "/product/123",
			wantRedirect: true,
			wantTarget:   "/item/123",
		},
		{
			name: "match regex host redirect",
			redirects: []*Redirect{
				{Type: RedirectTypeRegexHost, Source: "example.com/user/([0-9]+)", Target: "/profile/$1", Status: RedirectStatusMovedPermanent},
			},
			host:         "example.com",
			uri:          "/user/456",
			wantRedirect: true,
			wantTarget:   "/profile/456",
		},
		{
			name: "no match returns nil",
			redirects: []*Redirect{
				{Type: RedirectTypeBasic, Source: "/existing", Target: "/new", Status: RedirectStatusMovedPermanent},
			},
			host:         "example.com",
			uri:          "/non-existing",
			wantRedirect: false,
			wantTarget:   "",
		},
		{
			name: "basic host has priority over basic",
			redirects: []*Redirect{
				{Type: RedirectTypeBasic, Source: "/page", Target: "/basic-target", Status: RedirectStatusMovedPermanent},
				{Type: RedirectTypeBasicHost, Source: "example.com/page", Target: "/host-target", Status: RedirectStatusMovedPermanent},
			},
			host:         "example.com",
			uri:          "/page",
			wantRedirect: true,
			wantTarget:   "/host-target",
		},
		{
			name: "basic has priority over regex",
			redirects: []*Redirect{
				{Type: RedirectTypeRegex, Source: "/page", Target: "/regex-target", Status: RedirectStatusMovedPermanent},
				{Type: RedirectTypeBasic, Source: "/page", Target: "/basic-target", Status: RedirectStatusMovedPermanent},
			},
			host:         "example.com",
			uri:          "/page",
			wantRedirect: true,
			wantTarget:   "/basic-target",
		},
		{
			name: "regex host has priority over regex",
			redirects: []*Redirect{
				{Type: RedirectTypeRegex, Source: "/user/([0-9]+)", Target: "/regex-target/$1", Status: RedirectStatusMovedPermanent},
				{Type: RedirectTypeRegexHost, Source: "example.com/user/([0-9]+)", Target: "/host-target/$1", Status: RedirectStatusMovedPermanent},
			},
			host:         "example.com",
			uri:          "/user/123",
			wantRedirect: true,
			wantTarget:   "/host-target/123",
		},
		{
			name: "longer regex source has priority",
			redirects: []*Redirect{
				{Type: RedirectTypeRegex, Source: "/product/(.*)", Target: "/short", Status: RedirectStatusMovedPermanent},
				{Type: RedirectTypeRegex, Source: "/product/category/(.*)", Target: "/long", Status: RedirectStatusMovedPermanent},
			},
			host:         "example.com",
			uri:          "/product/category/shoes",
			wantRedirect: true,
			wantTarget:   "/long",
		},
		{
			name: "match with multiple capture groups",
			redirects: []*Redirect{
				{Type: RedirectTypeRegex, Source: "/([a-z]+)/([0-9]+)/([a-z]+)", Target: "/$3/$2/$1", Status: RedirectStatusMovedPermanent},
			},
			host:         "example.com",
			uri:          "/category/123/product",
			wantRedirect: true,
			wantTarget:   "/product/123/category",
		},
		{
			name: "match regex without prefix in root bucket",
			redirects: []*Redirect{
				{Type: RedirectTypeRegex, Source: ".*\\.html", Target: "/html-page", Status: RedirectStatusMovedPermanent},
			},
			host:         "example.com",
			uri:          "page.html",
			wantRedirect: true,
			wantTarget:   "/html-page",
		},
		{
			name: "match empty uri",
			redirects: []*Redirect{
				{Type: RedirectTypeBasic, Source: "", Target: "/home", Status: RedirectStatusMovedPermanent},
			},
			host:         "example.com",
			uri:          "",
			wantRedirect: true,
			wantTarget:   "/home",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := NewRedirectTreeMatcher()

			for _, r := range tt.redirects {
				assert.NoError(t, tree.Insert(r))
			}

			gotRedirect, gotTarget := tree.Match(tt.host, tt.uri)

			if tt.wantRedirect {
				assert.NotNil(t, gotRedirect)
			} else {
				assert.Nil(t, gotRedirect)
			}
			assert.Equal(t, tt.wantTarget, gotTarget)
		})
	}
}

func Test_resolveTarget(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		matches []string
		want    string
	}{
		{
			name:    "no placeholders",
			target:  "/static/path",
			matches: []string{"/full/match"},
			want:    "/static/path",
		},
		{
			name:    "single placeholder",
			target:  "/item/$1",
			matches: []string{"/product/123", "123"},
			want:    "/item/123",
		},
		{
			name:    "multiple placeholders",
			target:  "/$2/$1",
			matches: []string{"/category/product", "product", "category"},
			want:    "/category/product",
		},
		{
			name:    "placeholder not in matches",
			target:  "/item/$3",
			matches: []string{"/match", "group1"},
			want:    "/item/$3",
		},
		{
			name:    "repeated placeholder",
			target:  "/$1/$1",
			matches: []string{"/test/abc", "abc"},
			want:    "/abc/abc",
		},
		{
			name:    "empty matches",
			target:  "/path/$1",
			matches: []string{},
			want:    "/path/$1",
		},
		{
			name:    "nine capture groups",
			target:  "/$1$2$3$4$5$6$7$8$9",
			matches: []string{"full", "a", "b", "c", "d", "e", "f", "g", "h", "i"},
			want:    "/abcdefghi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveTarget(tt.target, tt.matches)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_extractRegexPrefix(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    string
	}{
		{
			name:    "literal path",
			pattern: "/product/item",
			want:    "/product/item",
		},
		{
			name:    "pattern with caret prefix",
			pattern: "^/product/item",
			want:    "/product/item",
		},
		{
			name:    "pattern with regex at end",
			pattern: "/product/[0-9]+",
			want:    "/product/",
		},
		{
			name:    "pattern starting with regex",
			pattern: "[a-z]+/item",
			want:    "",
		},
		{
			name:    "pattern with capture group",
			pattern: "/user/([0-9]+)/profile",
			want:    "/user/",
		},
		{
			name:    "empty pattern",
			pattern: "",
			want:    "",
		},
		{
			name:    "only regex characters",
			pattern: ".*",
			want:    "",
		},
		{
			name:    "invalid regex returns empty",
			pattern: "[invalid(regex",
			want:    "",
		},
		{
			name:    "complex pattern with literals and regex",
			pattern: "/api/v1/users/[0-9]+/posts",
			want:    "/api/v1/users/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractRegexPrefix(tt.pattern)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_extractLiteralPrefix(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    string
	}{
		{
			name:    "OpLiteral - simple string",
			pattern: "abc",
			want:    "abc",
		},
		{
			name:    "OpConcat - literal followed by regex",
			pattern: "/path/[0-9]+",
			want:    "/path/",
		},
		{
			name:    "OpConcat - multiple literals",
			pattern: "/api/v1/users",
			want:    "/api/v1/users",
		},
		{
			name:    "OpCapture - with literal inside",
			pattern: "(abc)def",
			want:    "abcdef",
		},
		{
			name:    "OpCapture - with regex inside",
			pattern: "([0-9]+)def",
			want:    "",
		},
		{
			name:    "OpConcat with capture containing literal",
			pattern: "/user/(profile)/settings",
			want:    "/user/profile/settings",
		},
		{
			name:    "starts with regex",
			pattern: "[a-z]+/path",
			want:    "",
		},
		{
			name:    "empty capture group",
			pattern: "()",
			want:    "",
		},
		{
			name:    "alternation breaks prefix",
			pattern: "/path/(a|b)/end",
			want:    "/path/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			re, err := syntax.Parse(tt.pattern, syntax.Perl)
			assert.NoError(t, err)

			got := extractLiteralPrefix(re)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_minInt(t *testing.T) {
	tests := []struct {
		name string
		a    int
		b    int
		want int
	}{
		{
			name: "a less than b",
			a:    1,
			b:    5,
			want: 1,
		},
		{
			name: "a greater than b",
			a:    10,
			b:    3,
			want: 3,
		},
		{
			name: "a equals b",
			a:    7,
			b:    7,
			want: 7,
		},
		{
			name: "negative numbers",
			a:    -5,
			b:    -2,
			want: -5,
		},
		{
			name: "zero and positive",
			a:    0,
			b:    5,
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := minInt(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_sortBySourceLength(t *testing.T) {
	tests := []struct {
		name       string
		candidates []*compiledRedirect
		wantOrder  []string
	}{
		{
			name:       "empty slice",
			candidates: []*compiledRedirect{},
			wantOrder:  []string{},
		},
		{
			name: "single element",
			candidates: []*compiledRedirect{
				{Redirect: &Redirect{Source: "/path"}},
			},
			wantOrder: []string{"/path"},
		},
		{
			name: "already sorted",
			candidates: []*compiledRedirect{
				{Redirect: &Redirect{Source: "/very/long/path"}},
				{Redirect: &Redirect{Source: "/medium"}},
				{Redirect: &Redirect{Source: "/a"}},
			},
			wantOrder: []string{"/very/long/path", "/medium", "/a"},
		},
		{
			name: "reverse order",
			candidates: []*compiledRedirect{
				{Redirect: &Redirect{Source: "/a"}},
				{Redirect: &Redirect{Source: "/medium"}},
				{Redirect: &Redirect{Source: "/very/long/path"}},
			},
			wantOrder: []string{"/very/long/path", "/medium", "/a"},
		},
		{
			name: "mixed order",
			candidates: []*compiledRedirect{
				{Redirect: &Redirect{Source: "/medium"}},
				{Redirect: &Redirect{Source: "/a"}},
				{Redirect: &Redirect{Source: "/very/long/path"}},
				{Redirect: &Redirect{Source: "/bb"}},
			},
			wantOrder: []string{"/very/long/path", "/medium", "/bb", "/a"},
		},
		{
			name: "same length elements preserve order",
			candidates: []*compiledRedirect{
				{Redirect: &Redirect{Source: "/aaa"}},
				{Redirect: &Redirect{Source: "/bbb"}},
				{Redirect: &Redirect{Source: "/ccc"}},
			},
			wantOrder: []string{"/aaa", "/bbb", "/ccc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sortBySourceLength(tt.candidates)

			assert.Len(t, tt.candidates, len(tt.wantOrder))

			for i, c := range tt.candidates {
				assert.Equal(t, tt.wantOrder[i], c.Source, "position %d", i)
			}
		})
	}
}

func TestRedirectTree_matchRegex(t *testing.T) {
	tests := []struct {
		name         string
		redirects    []*Redirect
		input        string
		wantRedirect bool
		wantTarget   string
	}{
		{
			name: "match from tree bucket",
			redirects: []*Redirect{
				{Type: RedirectTypeRegex, Source: "/product/([0-9]+)", Target: "/item/$1", Status: RedirectStatusMovedPermanent},
			},
			input:        "/product/123",
			wantRedirect: true,
			wantTarget:   "/item/123",
		},
		{
			name: "match from root bucket",
			redirects: []*Redirect{
				{Type: RedirectTypeRegex, Source: "([a-z]+)\\.html", Target: "/$1", Status: RedirectStatusMovedPermanent},
			},
			input:        "page.html",
			wantRedirect: true,
			wantTarget:   "/page",
		},
		{
			name: "no match",
			redirects: []*Redirect{
				{Type: RedirectTypeRegex, Source: "/product/([0-9]+)", Target: "/item/$1", Status: RedirectStatusMovedPermanent},
			},
			input:        "/category/abc",
			wantRedirect: false,
			wantTarget:   "",
		},
		{
			name: "longer pattern wins",
			redirects: []*Redirect{
				{Type: RedirectTypeRegex, Source: "/a/(.*)", Target: "/short", Status: RedirectStatusMovedPermanent},
				{Type: RedirectTypeRegex, Source: "/a/b/(.*)", Target: "/long", Status: RedirectStatusMovedPermanent},
			},
			input:        "/a/b/c",
			wantRedirect: true,
			wantTarget:   "/long",
		},
		{
			name:         "empty redirects and input",
			redirects:    []*Redirect{},
			input:        "",
			wantRedirect: false,
			wantTarget:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := NewRedirectTreeMatcher().(*RedirectTree)

			for _, r := range tt.redirects {
				assert.NoError(t, tree.Insert(r))
			}

			gotRedirect, gotTarget := tree.matchRegex(tree.regex, tree.regexRoot, tt.input)

			if tt.wantRedirect {
				assert.NotNil(t, gotRedirect)
			} else {
				assert.Nil(t, gotRedirect)
			}
			assert.Equal(t, tt.wantTarget, gotTarget)
		})
	}
}
