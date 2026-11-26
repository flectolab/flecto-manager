package types

import (
	"encoding/json"
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
func TestRedirectOptions_IsIgnoreQueryString(t *testing.T) {
	tests := []struct {
		name    string
		options RedirectOptions
		want    bool
	}{
		{name: "not set", options: RedirectOptions{}, want: false},
		{name: "false", options: RedirectOptions{IgnoreQueryString: boolPtr(false)}, want: false},
		{name: "true", options: RedirectOptions{IgnoreQueryString: boolPtr(true)}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.options.IsIgnoreQueryString())
		})
	}
}

func TestRedirectOptions_IsPreserveQueryString(t *testing.T) {
	tests := []struct {
		name    string
		options RedirectOptions
		want    bool
	}{
		{name: "not set", options: RedirectOptions{}, want: false},
		{name: "false", options: RedirectOptions{PreserveQueryString: boolPtr(false)}, want: false},
		{name: "true", options: RedirectOptions{PreserveQueryString: boolPtr(true)}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.options.IsPreserveQueryString())
		})
	}
}

func TestRedirectOptions_Equal(t *testing.T) {
	tests := []struct {
		name string
		a    RedirectOptions
		b    RedirectOptions
		want bool
	}{
		{name: "both empty", a: RedirectOptions{}, b: RedirectOptions{}, want: true},
		{name: "same values different pointers", a: RedirectOptions{IgnoreQueryString: boolPtr(true)}, b: RedirectOptions{IgnoreQueryString: boolPtr(true)}, want: true},
		{name: "nil and false differ", a: RedirectOptions{}, b: RedirectOptions{IgnoreQueryString: boolPtr(false)}, want: false},
		{name: "true and false differ", a: RedirectOptions{PreserveQueryString: boolPtr(true)}, b: RedirectOptions{PreserveQueryString: boolPtr(false)}, want: false},
		{name: "other option differs", a: RedirectOptions{IgnoreQueryString: boolPtr(true)}, b: RedirectOptions{IgnoreQueryString: boolPtr(true), PreserveQueryString: boolPtr(true)}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.a.Equal(tt.b))
			assert.Equal(t, tt.want, tt.b.Equal(tt.a))
		})
	}
}

func TestRedirectOptions_JSON(t *testing.T) {
	tests := []struct {
		name    string
		options RedirectOptions
		want    string
	}{
		{name: "not set is omitted", options: RedirectOptions{}, want: `{}`},
		{name: "false is kept", options: RedirectOptions{IgnoreQueryString: boolPtr(false)}, want: `{"ignoreQueryString":false}`},
		{
			name:    "both set",
			options: RedirectOptions{IgnoreQueryString: boolPtr(true), PreserveQueryString: boolPtr(true)},
			want:    `{"ignoreQueryString":true,"preserveQueryString":true}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.options)
			assert.NoError(t, err)
			assert.JSONEq(t, tt.want, string(got))

			var decoded RedirectOptions
			assert.NoError(t, json.Unmarshal(got, &decoded))
			assert.Equal(t, tt.options, decoded)
		})
	}
}

func TestRedirect_JSONWithoutOptions(t *testing.T) {
	var r Redirect
	err := json.Unmarshal([]byte(`{"type":"BASIC","source":"/a","target":"/b","status":"MOVED_PERMANENT"}`), &r)

	assert.NoError(t, err)
	assert.Equal(t, RedirectOptions{}, r.Options)
}

func TestAppendQueryString(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		rawQuery string
		want     string
	}{
		{name: "empty query string", target: "/b", rawQuery: "", want: "/b"},
		{name: "plain target", target: "/b", rawQuery: "utm_source=nl", want: "/b?utm_source=nl"},
		{name: "query string kept raw", target: "/b", rawQuery: "q=a%20b&x", want: "/b?q=a%20b&x"},
		{name: "absolute target", target: "https://example.com/b", rawQuery: "utm_source=nl", want: "https://example.com/b?utm_source=nl"},
		{name: "before the fragment", target: "/b#top", rawQuery: "utm_source=nl", want: "/b?utm_source=nl#top"},
		{name: "target with empty query string", target: "/b?", rawQuery: "utm_source=nl", want: "/b?utm_source=nl"},
		{name: "merged with the target query string", target: "/b?lang=fr", rawQuery: "utm_source=nl", want: "/b?lang=fr&utm_source=nl"},
		{name: "target wins on a shared key", target: "/b?lang=fr", rawQuery: "lang=en&utm_source=nl", want: "/b?lang=fr&utm_source=nl"},
		{name: "target query kept as is", target: "/b?next=/x&amp&z=a+b", rawQuery: "utm_source=nl", want: "/b?next=/x&amp&z=a+b&utm_source=nl"},
		{name: "request params kept raw and in order", target: "/b?lang=fr", rawQuery: "z=2&a=%2F&flag", want: "/b?lang=fr&z=2&a=%2F&flag"},
		{name: "encoded request key matches the target key", target: "/b?utm_source=x", rawQuery: "utm%5Fsource=nl&y=1", want: "/b?utm_source=x&y=1"},
		{name: "every request param already in the target", target: "/b?lang=fr", rawQuery: "lang=en", want: "/b?lang=fr"},
		{name: "empty request params skipped", target: "/b?lang=fr", rawQuery: "&utm_source=nl&", want: "/b?lang=fr&utm_source=nl"},
		{name: "merged before the fragment", target: "/b?lang=fr#top", rawQuery: "utm_source=nl", want: "/b?lang=fr&utm_source=nl#top"},
		{name: "undecodable request query appended raw", target: "/b?lang=fr", rawQuery: "x=%zz", want: "/b?lang=fr&x=%zz"},
		{name: "undecodable target query appended raw", target: "/b?x=%zz", rawQuery: "utm_source=nl", want: "/b?x=%zz&utm_source=nl"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, AppendQueryString(tt.target, tt.rawQuery))
		})
	}
}
