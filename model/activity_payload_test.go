package model

import (
	"encoding/json"
	"testing"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
	"github.com/stretchr/testify/assert"
)

func TestNewRedirectSnapshot(t *testing.T) {
	t.Run("projects every field", func(t *testing.T) {
		snapshot := NewRedirectSnapshot(&commonTypes.Redirect{
			Type:   commonTypes.RedirectTypeBasic,
			Source: "/old",
			Target: "/new",
			Status: commonTypes.RedirectStatusMovedPermanent,
		})

		assert.Equal(t, commonTypes.RedirectTypeBasic, snapshot.Type)
		assert.Equal(t, "/old", snapshot.Source)
		assert.Equal(t, "/new", snapshot.Target)
		assert.Equal(t, commonTypes.RedirectStatusMovedPermanent, snapshot.Status)
	})

	t.Run("nil in, nil out", func(t *testing.T) {
		assert.Nil(t, NewRedirectSnapshot(nil))
	})
}

func TestNewPageSnapshot(t *testing.T) {
	t.Run("leaves the content out", func(t *testing.T) {
		snapshot := NewPageSnapshot(&commonTypes.Page{
			Type:        commonTypes.PageTypeBasic,
			Path:        "/robots.txt",
			Content:     "User-agent: *",
			ContentType: commonTypes.PageContentTypeTextPlain,
		}, 13)

		assert.Equal(t, commonTypes.PageTypeBasic, snapshot.Type)
		assert.Equal(t, "/robots.txt", snapshot.Path)
		assert.Equal(t, commonTypes.PageContentTypeTextPlain, snapshot.ContentType)
		assert.Equal(t, int64(13), snapshot.ContentSize)

		out, err := json.Marshal(snapshot)
		assert.NoError(t, err)
		assert.NotContains(t, string(out), "User-agent")
	})

	t.Run("nil in, nil out", func(t *testing.T) {
		assert.Nil(t, NewPageSnapshot(nil, 0))
	})
}

func TestActivityPayloads_JSONShape(t *testing.T) {
	t.Run("change omits the missing side", func(t *testing.T) {
		out, err := json.Marshal(ActivityChange[RedirectSnapshot]{
			After: &RedirectSnapshot{Source: "/a", Target: "/b"},
		})
		assert.NoError(t, err)
		assert.NotContains(t, string(out), "before")
		assert.Contains(t, string(out), "after")
	})

	t.Run("project rollback omits the single-entry fields", func(t *testing.T) {
		out, err := json.Marshal(ActivityRollback[RedirectSnapshot]{
			Scope:     ActivityRollbackScopeProject,
			Discarded: &ActivityDraftCounts{Create: 2, Update: 1},
		})
		assert.NoError(t, err)
		assert.JSONEq(t, `{"scope":"PROJECT","discarded":{"create":2,"update":1,"delete":0}}`, string(out))
	})

	t.Run("single rollback carries the entry", func(t *testing.T) {
		changeType := DraftChangeTypeCreate
		out, err := json.Marshal(ActivityRollback[RedirectSnapshot]{
			Scope:      ActivityRollbackScopeSingle,
			ChangeType: &changeType,
			Entry:      &RedirectSnapshot{Source: "/a"},
		})
		assert.NoError(t, err)
		assert.Contains(t, string(out), `"scope":"SINGLE"`)
		assert.Contains(t, string(out), `"changeType":"CREATE"`)
		assert.NotContains(t, string(out), "discarded")
	})
}
