package cdn

import (
	"testing"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
	"github.com/flectolab/flecto-manager/types"
	"github.com/stretchr/testify/assert"
)

func TestRedirectEntry(t *testing.T) {
	tests := []struct {
		name     string
		redirect *commonTypes.Redirect
		want     Entry
	}{
		{
			name:     "successWithBasic",
			redirect: &commonTypes.Redirect{Type: commonTypes.RedirectTypeBasic, Source: "/promo", Target: "/new"},
			want:     Entry{Type: "BASIC", Source: "/promo"},
		},
		{
			name:     "successWithBasicHost",
			redirect: &commonTypes.Redirect{Type: commonTypes.RedirectTypeBasicHost, Source: "example.com/promo"},
			want:     Entry{Type: "BASIC_HOST", Source: "example.com/promo"},
		},
		{
			name: "successWithIgnoreQueryString",
			redirect: &commonTypes.Redirect{
				Type:    commonTypes.RedirectTypeBasic,
				Source:  "/promo",
				Options: commonTypes.RedirectOptions{IgnoreQueryString: types.Ptr(true)},
			},
			want: Entry{Type: "BASIC", Source: "/promo", IgnoreQueryString: true},
		},
		{
			name:     "successWithNil",
			redirect: nil,
			want:     Entry{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, RedirectEntry(tt.redirect))
		})
	}
}

func TestPageEntry(t *testing.T) {
	tests := []struct {
		name string
		page *commonTypes.Page
		want Entry
	}{
		{
			name: "successWithBasic",
			page: &commonTypes.Page{Type: commonTypes.PageTypeBasic, Path: "/robots.txt"},
			want: Entry{Type: "BASIC", Source: "/robots.txt"},
		},
		{
			name: "successWithBasicHost",
			page: &commonTypes.Page{Type: commonTypes.PageTypeBasicHost, Path: "example.com/robots.txt"},
			want: Entry{Type: "BASIC_HOST", Source: "example.com/robots.txt"},
		},
		{
			name: "successWithNil",
			page: nil,
			want: Entry{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, PageEntry(tt.page))
		})
	}
}

func TestEntry_Resolve(t *testing.T) {
	tests := []struct {
		name     string
		entry    Entry
		want     Location
		wantSkip SkipReason
	}{
		{
			name:  "successWithBasic",
			entry: Entry{Type: "BASIC", Source: "/promo"},
			want:  Location{Path: "/promo"},
		},
		{
			name:  "successWithBasicHost",
			entry: Entry{Type: "BASIC_HOST", Source: "example.com/promo"},
			want:  Location{Host: "example.com", Path: "/promo"},
		},
		{
			name:  "successLowercasesHost",
			entry: Entry{Type: "BASIC_HOST", Source: "Example.COM/promo"},
			want:  Location{Host: "example.com", Path: "/promo"},
		},
		{
			name:  "successKeepsPort",
			entry: Entry{Type: "BASIC_HOST", Source: "example.com:8080/promo"},
			want:  Location{Host: "example.com:8080", Path: "/promo"},
		},
		{
			name:  "successKeepsTheQueryString",
			entry: Entry{Type: "BASIC_HOST", Source: "example.com/promo?a=1"},
			want:  Location{Host: "example.com", Path: "/promo?a=1"},
		},
		{
			name:  "successKeepsThePathEncoded",
			entry: Entry{Type: "BASIC_HOST", Source: "example.com/caf%C3%A9"},
			want:  Location{Host: "example.com", Path: "/caf%C3%A9"},
		},
		{
			name:     "skipWithRegex",
			entry:    Entry{Type: "REGEX", Source: "^/promo/(.*)$"},
			wantSkip: SkipReasonPattern,
		},
		{
			name:     "skipWithRegexHost",
			entry:    Entry{Type: "REGEX_HOST", Source: "^example.com/promo/(.*)$"},
			wantSkip: SkipReasonPattern,
		},
		{
			name:     "skipWithBasicNotRooted",
			entry:    Entry{Type: "BASIC", Source: "promo"},
			wantSkip: SkipReasonInvalid,
		},
		{
			name:     "skipWithBasicHostWithoutPath",
			entry:    Entry{Type: "BASIC_HOST", Source: "example.com"},
			wantSkip: SkipReasonInvalid,
		},
		{
			name:     "skipWithBasicHostWithoutHost",
			entry:    Entry{Type: "BASIC_HOST", Source: "/promo"},
			wantSkip: SkipReasonInvalid,
		},
		{
			name:     "skipWithUnknownType",
			entry:    Entry{Type: "SOMETHING", Source: "/promo"},
			wantSkip: SkipReasonInvalid,
		},
		{
			name:     "skipWithEmptyEntry",
			entry:    Entry{},
			wantSkip: SkipReasonInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, skip := tt.entry.Resolve()

			assert.Equal(t, tt.wantSkip, skip)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewPlanner(t *testing.T) {
	t.Run("successNormalizesAndSorts", func(t *testing.T) {
		planner := NewPlanner([]string{" WWW.example.com ", "a.example.com", "www.example.com", ""})

		assert.Equal(t, []string{"a.example.com", "www.example.com"}, planner.Hosts())
	})

	t.Run("successKeepsWildcardsOutOfHosts", func(t *testing.T) {
		planner := NewPlanner([]string{"*.example.com", "www.example.com"})

		// A wildcard routes host-carrying entries but never expands host-less ones.
		assert.Equal(t, []string{"www.example.com"}, planner.Hosts())
	})
}

func TestPlanner_Serves(t *testing.T) {
	planner := NewPlanner([]string{"www.example.com", "*.shop.example.com"})

	tests := []struct {
		name string
		host string
		want bool
	}{
		{name: "successWithExactHost", host: "www.example.com", want: true},
		{name: "successWithDifferentCase", host: "WWW.example.com", want: true},
		{name: "successWithWildcardSubdomain", host: "fr.shop.example.com", want: true},
		{name: "successWithDeepWildcardSubdomain", host: "a.b.shop.example.com", want: true},
		{name: "failWithWildcardApex", host: "shop.example.com", want: false},
		{name: "failWithUnknownHost", host: "other.com", want: false},
		{name: "failWithEmptyHost", host: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, planner.Serves(tt.host))
		})
	}
}

func TestPlanner_Build(t *testing.T) {
	tests := []struct {
		name         string
		hosts        []string
		capabilities Capabilities
		entries      []Entry
		wantTargets  []Target
		wantSkipped  map[SkipReason]int
	}{
		{
			name:         "successPurgesByPrefixAnEntryIgnoringQueryString",
			hosts:        []string{"www.example.com", "example.com"},
			capabilities: Capabilities{URL: true, Prefix: true},
			entries: []Entry{
				{Type: "BASIC", Source: "/promo", IgnoreQueryString: true},
				{Type: "BASIC", Source: "/other"},
			},
			wantTargets: []Target{
				{Kind: KindPrefix, Host: "example.com", Path: "/promo"},
				{Kind: KindPrefix, Host: "www.example.com", Path: "/promo"},
				{Kind: KindURL, Host: "example.com", Path: "/other"},
				{Kind: KindURL, Host: "www.example.com", Path: "/other"},
			},
			wantSkipped: map[SkipReason]int{},
		},
		{
			name:         "successPurgesByUrlAnEntryIgnoringQueryStringWithoutPrefixPurge",
			hosts:        []string{"www.example.com"},
			capabilities: Capabilities{URL: true},
			entries:      []Entry{{Type: "BASIC_HOST", Source: "www.example.com/promo", IgnoreQueryString: true}},
			wantTargets: []Target{
				{Kind: KindURL, Host: "www.example.com", Path: "/promo"},
			},
			wantSkipped: map[SkipReason]int{SkipReasonQueryVariants: 1},
		},
		{
			name:         "skipsOnceAnEntryIgnoringQueryStringWhoseHostIsNotServed",
			hosts:        []string{"www.example.com"},
			capabilities: Capabilities{URL: true},
			entries:      []Entry{{Type: "BASIC_HOST", Source: "other.com/promo", IgnoreQueryString: true}},
			wantSkipped:  map[SkipReason]int{SkipReasonNoHostMatch: 1},
		},
		{
			name:    "successExpandsHostLessEntryOverEveryHost",
			hosts:   []string{"www.example.com", "example.com"},
			entries: []Entry{{Type: "BASIC", Source: "/promo"}},
			wantTargets: []Target{
				{Kind: KindURL, Host: "example.com", Path: "/promo"},
				{Kind: KindURL, Host: "www.example.com", Path: "/promo"},
			},
			wantSkipped: map[SkipReason]int{},
		},
		{
			name:    "successRoutesEntryCarryingItsHost",
			hosts:   []string{"www.example.com", "shop.example.com"},
			entries: []Entry{{Type: "BASIC_HOST", Source: "shop.example.com/soldes"}},
			wantTargets: []Target{
				{Kind: KindURL, Host: "shop.example.com", Path: "/soldes"},
			},
			wantSkipped: map[SkipReason]int{},
		},
		{
			name:        "skipsEntryWhoseHostIsNotServed",
			hosts:       []string{"www.example.com"},
			entries:     []Entry{{Type: "BASIC_HOST", Source: "other.com/soldes"}},
			wantSkipped: map[SkipReason]int{SkipReasonNoHostMatch: 1},
		},
		{
			name:        "skipsHostLessEntryWithoutConcreteHost",
			hosts:       []string{"*.example.com"},
			entries:     []Entry{{Type: "BASIC", Source: "/promo"}},
			wantSkipped: map[SkipReason]int{SkipReasonNoHost: 1},
		},
		{
			name:  "skipsPatternEntries",
			hosts: []string{"www.example.com"},
			entries: []Entry{
				{Type: "REGEX", Source: "^/promo/(.*)$"},
				{Type: "REGEX_HOST", Source: "^www.example.com/a/(.*)$"},
			},
			wantSkipped: map[SkipReason]int{SkipReasonPattern: 2},
		},
		{
			name:  "successDeduplicatesTargets",
			hosts: []string{"www.example.com"},
			entries: []Entry{
				{Type: "BASIC", Source: "/promo"},
				{Type: "BASIC", Source: "/promo"},
				{Type: "BASIC_HOST", Source: "www.example.com/promo"},
			},
			wantTargets: []Target{
				{Kind: KindURL, Host: "www.example.com", Path: "/promo"},
			},
			wantSkipped: map[SkipReason]int{},
		},
		{
			name:  "successSortsTargets",
			hosts: []string{"b.example.com", "a.example.com"},
			entries: []Entry{
				{Type: "BASIC", Source: "/z"},
				{Type: "BASIC", Source: "/a"},
			},
			wantTargets: []Target{
				{Kind: KindURL, Host: "a.example.com", Path: "/a"},
				{Kind: KindURL, Host: "a.example.com", Path: "/z"},
				{Kind: KindURL, Host: "b.example.com", Path: "/a"},
				{Kind: KindURL, Host: "b.example.com", Path: "/z"},
			},
			wantSkipped: map[SkipReason]int{},
		},
		{
			name:  "successCountsEveryReasonSeparately",
			hosts: []string{"www.example.com"},
			entries: []Entry{
				{Type: "BASIC", Source: "/promo"},
				{Type: "REGEX", Source: "^/a"},
				{Type: "BASIC_HOST", Source: "other.com/b"},
				{Type: "BASIC", Source: "not-rooted"},
			},
			wantTargets: []Target{
				{Kind: KindURL, Host: "www.example.com", Path: "/promo"},
			},
			wantSkipped: map[SkipReason]int{
				SkipReasonPattern:     1,
				SkipReasonNoHostMatch: 1,
				SkipReasonInvalid:     1,
			},
		},
		{
			name:  "successPurgesBothFormsOfAnEncodedPath",
			hosts: []string{"www.example.com"},
			entries: []Entry{
				{Type: "BASIC", Source: "/café"},
				{Type: "BASIC_HOST", Source: "www.example.com/caf%C3%A9"},
			},
			wantTargets: []Target{
				{Kind: KindURL, Host: "www.example.com", Path: "/caf%C3%A9"},
				{Kind: KindURL, Host: "www.example.com", Path: "/café"},
			},
			wantSkipped: map[SkipReason]int{},
		},
		{
			name:        "successWithoutEntries",
			hosts:       []string{"www.example.com"},
			entries:     nil,
			wantSkipped: map[SkipReason]int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewPlanner(tt.hosts).WithCapabilities(tt.capabilities).Build(tt.entries)

			assert.Equal(t, tt.wantTargets, got.Targets)
			assert.Equal(t, tt.wantSkipped, got.Skipped)
		})
	}
}

func TestPathVariants(t *testing.T) {
	tests := []struct {
		name string
		path string
		want []string
	}{
		{name: "successWithAPlainPath", path: "/promo", want: []string{"/promo"}},
		{name: "successWithAnEncodedPath", path: "/caf%C3%A9", want: []string{"/caf%C3%A9", "/café"}},
		{name: "successWithADecodedPath", path: "/café", want: []string{"/caf%C3%A9", "/café"}},
		{name: "successKeepsTheQueryString", path: "/caf%C3%A9?a=1", want: []string{"/caf%C3%A9?a=1", "/café?a=1"}},
		{name: "successKeepsAPathStartingWithTwoSlashes", path: "//promo", want: []string{"//promo"}},
		{name: "successWithoutDecodingASpace", path: "/a%20b", want: []string{"/a%20b"}},
		{name: "successWithoutDecodingAPercent", path: "/100%25", want: []string{"/100%25"}},
		{name: "successWithoutDecodingAQuestionMark", path: "/a%3Fb", want: []string{"/a%3Fb"}},
		{name: "successWithAnUnparsablePath", path: "/%zz", want: []string{"/%zz"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, PathVariants(tt.path))
		})
	}
}

func TestPlan_IsEmpty(t *testing.T) {
	assert.True(t, Plan{}.IsEmpty())
	assert.False(t, Plan{Targets: []Target{{Kind: KindURL}}}.IsEmpty())
}

func TestPlan_SkippedTotal(t *testing.T) {
	plan := Plan{Skipped: map[SkipReason]int{SkipReasonPattern: 2, SkipReasonNoHostMatch: 3}}

	assert.Equal(t, 5, plan.SkippedTotal())
	assert.Equal(t, 0, Plan{}.SkippedTotal())
}

func TestBatch(t *testing.T) {
	targets := []Target{
		{Kind: KindURL, Path: "/1"},
		{Kind: KindURL, Path: "/2"},
		{Kind: KindURL, Path: "/3"},
	}
	mixed := []Target{
		{Kind: KindPrefix, Path: "/p1"},
		{Kind: KindPrefix, Path: "/p2"},
		{Kind: KindURL, Path: "/1"},
		{Kind: KindURL, Path: "/2"},
		{Kind: KindURL, Path: "/3"},
	}

	tests := []struct {
		name    string
		targets []Target
		limits  Limits
		want    [][]Target
	}{
		{
			name:    "successWithExactMultiple",
			targets: targets[:2],
			limits:  Limits{MaxTargetsPerRequest: 2},
			want:    [][]Target{targets[:2]},
		},
		{
			name:    "successWithRemainder",
			targets: targets,
			limits:  Limits{MaxTargetsPerRequest: 2},
			want:    [][]Target{targets[:2], targets[2:]},
		},
		{
			name:    "successWithSizeLargerThanInput",
			targets: targets,
			limits:  Limits{MaxTargetsPerRequest: 10},
			want:    [][]Target{targets},
		},
		{
			name:    "successWithoutSizeYieldsSingleBatch",
			targets: targets,
			limits:  Limits{},
			want:    [][]Target{targets},
		},
		{
			name:    "successWithoutTargets",
			targets: nil,
			limits:  Limits{MaxTargetsPerRequest: 2},
			want:    nil,
		},
		{
			name:    "successCutsOnKindChange",
			targets: mixed,
			limits:  Limits{MaxTargetsPerRequest: 10},
			want:    [][]Target{mixed[:2], mixed[2:]},
		},
		{
			name:    "successUsesTheSizeOfEachKind",
			targets: mixed,
			limits: Limits{
				MaxTargetsPerRequest: 2,
				ByKind:               map[TargetKind]Limits{KindPrefix: {MaxTargetsPerRequest: 1}},
			},
			want: [][]Target{mixed[:1], mixed[1:2], mixed[2:4], mixed[4:]},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Batch(tt.targets, tt.limits))
		})
	}
}

func TestLimits_For(t *testing.T) {
	prefix := Limits{MaxTargetsPerRequest: 100, RequestsPerSecond: 10, Burst: 50}
	limits := Limits{MaxTargetsPerRequest: 500, TargetsPerSecond: 3000, ByKind: map[TargetKind]Limits{KindPrefix: prefix}}

	assert.Equal(t, prefix, limits.For(KindPrefix))
	assert.Equal(t, Limits{MaxTargetsPerRequest: 500, TargetsPerSecond: 3000}, limits.For(KindURL))
}

func TestSortTargets(t *testing.T) {
	targets := []Target{
		{Kind: KindURL, Host: "b.com", Path: "/a"},
		{Kind: KindPrefix, Host: "b.com", Path: "/z"},
		{Kind: KindURL, Host: "a.com", Path: "/b"},
		{Kind: KindURL, Host: "a.com", Path: "/a"},
	}

	SortTargets(targets)

	assert.Equal(t, []Target{
		{Kind: KindPrefix, Host: "b.com", Path: "/z"},
		{Kind: KindURL, Host: "a.com", Path: "/a"},
		{Kind: KindURL, Host: "a.com", Path: "/b"},
		{Kind: KindURL, Host: "b.com", Path: "/a"},
	}, targets)
}
