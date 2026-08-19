package model

import (
	commonTypes "github.com/flectolab/flecto-manager/common/types"
)

// ActivityImportErrorSampleMax bounds the errors kept in an import payload. The
// payload keeps an exact ErrorCount, so a truncated sample is detectable with
// errorCount > len(errorSample).
const ActivityImportErrorSampleMax = 20

// ActivityInput describes an event to record. The actor and the timestamp are filled
// in by the activity service from the request context.
//
// It lives in model rather than service so that the generated service mocks stay
// free of any import back into service, which would make the service tests cycle.
type ActivityInput struct {
	NamespaceCode string
	ProjectCode   string
	Resource      ActivityResource
	Action        ActivityAction
	ResourceID    *int64
	// Data is the payload, marshalled to JSON when recorded. Its type must match
	// the (Resource, Action) pair, see the payload types below.
	Data any
}

// RedirectSnapshot is the activity projection of a redirect.
type RedirectSnapshot struct {
	Type   commonTypes.RedirectType   `json:"type"`
	Source string                     `json:"source"`
	Target string                     `json:"target"`
	Status commonTypes.RedirectStatus `json:"status"`
}

// NewRedirectSnapshot projects a redirect for the activity trail, nil in, nil out.
func NewRedirectSnapshot(redirect *commonTypes.Redirect) *RedirectSnapshot {
	if redirect == nil {
		return nil
	}
	return &RedirectSnapshot{
		Type:   redirect.Type,
		Source: redirect.Source,
		Target: redirect.Target,
		Status: redirect.Status,
	}
}

// PageSnapshot is the activity projection of a page. The page content is
// deliberately left out: it can reach page.size_limit (1MB by default), which has
// no place in a journal kept for months.
type PageSnapshot struct {
	Type        commonTypes.PageType        `json:"type"`
	Path        string                      `json:"path"`
	ContentType commonTypes.PageContentType `json:"contentType"`
	ContentSize int64                       `json:"contentSize"`
}

// NewPageSnapshot projects a page for the activity trail, nil in, nil out.
func NewPageSnapshot(page *commonTypes.Page, contentSize int64) *PageSnapshot {
	if page == nil {
		return nil
	}
	return &PageSnapshot{
		Type:        page.Type,
		Path:        page.Path,
		ContentType: page.ContentType,
		ContentSize: contentSize,
	}
}

// ActivityChange carries the before and after of a single-entry change. Before is nil
// on a creation, After is nil on a deletion.
type ActivityChange[T any] struct {
	Before *T `json:"before,omitempty"`
	After  *T `json:"after,omitempty"`
}

// ActivityRollbackScope tells a rollback of the whole project apart from the
// cancellation of one pending change.
type ActivityRollbackScope string

const (
	ActivityRollbackScopeSingle  ActivityRollbackScope = "SINGLE"
	ActivityRollbackScopeProject ActivityRollbackScope = "PROJECT"
)

// ActivityRollback describes discarded pending changes. SINGLE fills ChangeType and
// Entry, PROJECT fills Discarded.
type ActivityRollback[T any] struct {
	Scope      ActivityRollbackScope `json:"scope"`
	ChangeType *DraftChangeType      `json:"changeType,omitempty"`
	Entry      *T                    `json:"entry,omitempty"`
	Discarded  *ActivityDraftCounts  `json:"discarded,omitempty"`
}

// ActivityDraftCounts breaks discarded drafts down by change type.
type ActivityDraftCounts struct {
	Create int64 `json:"create"`
	Update int64 `json:"update"`
	Delete int64 `json:"delete"`
}

// IsEmpty reports whether nothing was discarded, in which case the rollback is a
// no-op and deserves no journal entry.
func (c *ActivityDraftCounts) IsEmpty() bool {
	return c == nil || c.Create+c.Update+c.Delete == 0
}

// ActivityImport summarises a redirect import as a single event, whatever the size
// of the imported file.
type ActivityImport struct {
	Filename    string                `json:"filename,omitempty"`
	Overwrite   bool                  `json:"overwrite"`
	TotalLines  int                   `json:"totalLines"`
	Imported    int                   `json:"imported"`
	Skipped     int                   `json:"skipped"`
	ErrorCount  int                   `json:"errorCount"`
	ErrorSample []ActivityImportError `json:"errorSample,omitempty"`
}

type ActivityImportError struct {
	Line   int    `json:"line"`
	Source string `json:"source,omitempty"`
	Reason string `json:"reason"`
}

// ActivityPublish summarises a publish: the version it produced and what it moved.
type ActivityPublish struct {
	Version   int                   `json:"version"`
	Redirects ActivityPublishCounts `json:"redirects"`
	Pages     ActivityPublishCounts `json:"pages"`
}

type ActivityPublishCounts struct {
	Created int `json:"created"`
	Updated int `json:"updated"`
	Deleted int `json:"deleted"`
}

// ActivityTruncate summarises wiping a resource of a project. Version is the project
// version the wipe produced, since removing every redirect or page has to be
// published for the agents to stop serving what was deleted.
type ActivityTruncate struct {
	Published int64 `json:"published"`
	Drafts    int64 `json:"drafts"`
	Version   int   `json:"version,omitempty"`
}
