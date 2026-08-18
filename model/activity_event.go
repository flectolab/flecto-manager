package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
	appTypes "github.com/flectolab/flecto-manager/types"
)

// ActivityResource is what an activity event is about.
type ActivityResource string

// ActivityAction is what was done to the resource. The pair (ActivityResource,
// ActivityAction) determines the shape of the event payload, see activity_payload.go.
type ActivityAction string

const (
	ActivityResourceRedirect ActivityResource = "REDIRECT"
	ActivityResourcePage     ActivityResource = "PAGE"
	ActivityResourceProject  ActivityResource = "PROJECT"
	// ActivityResourceActivity is the journal itself, which can be cleared.
	ActivityResourceActivity ActivityResource = "ACTIVITY"

	ActivityActionCreate   ActivityAction = "CREATE"
	ActivityActionUpdate   ActivityAction = "UPDATE"
	ActivityActionDelete   ActivityAction = "DELETE"
	ActivityActionImport   ActivityAction = "IMPORT"
	ActivityActionRollback ActivityAction = "ROLLBACK"
	ActivityActionPublish  ActivityAction = "PUBLISH"
	// ActivityActionTruncate wipes every entry of a resource in a project.
	ActivityActionTruncate ActivityAction = "TRUNCATE"
)

// ActivityActorSystem is recorded when no authenticated subject is in the context,
// for instance a CLI command.
const ActivityActorSystem = "system"

var ActivityEventSortableColumns = map[string]string{
	"occurredAt": "occurred_at",
	"resource":   "resource",
	"action":     "action",
	"actor":      "actor",
}

// ErrActivityDataReadOnly is returned when a client tries to supply an activity payload.
var ErrActivityDataReadOnly = errors.New("activity data is read-only")

// ActivityData is a raw JSON payload written by the server only. Its shape is given
// by the (Resource, Action) pair of the event carrying it, and the front-end picks
// a renderer from that same pair.
type ActivityData json.RawMessage

func (d ActivityData) Value() (driver.Value, error) {
	if len(d) == 0 {
		return nil, nil
	}
	return string(d), nil
}

func (d *ActivityData) Scan(value any) error {
	switch v := value.(type) {
	case nil:
		*d = nil
	case []byte:
		*d = append(ActivityData{}, v...)
	case string:
		*d = ActivityData(v)
	default:
		return fmt.Errorf("cannot scan %T into ActivityData", value)
	}
	return nil
}

// MarshalJSON emits the payload as-is. Without it the underlying []byte would be
// base64 encoded.
func (d ActivityData) MarshalJSON() ([]byte, error) {
	if len(d) == 0 {
		return []byte("null"), nil
	}
	return d, nil
}

func (d *ActivityData) UnmarshalJSON(data []byte) error {
	*d = append(ActivityData{}, data...)
	return nil
}

// MarshalGQL writes the payload straight into the response: it already holds JSON
// produced server-side.
func (d ActivityData) MarshalGQL(w io.Writer) {
	if len(d) == 0 {
		_, _ = w.Write([]byte("null"))
		return
	}
	_, _ = w.Write(d)
}

// UnmarshalGQL always fails: activity payloads are never provided by clients.
func (d *ActivityData) UnmarshalGQL(any) error {
	return ErrActivityDataReadOnly
}

// ActivityEvent records one user action on a project. Rows are immutable: they are
// never updated, only inserted and eventually purged.
type ActivityEvent struct {
	ID            int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	NamespaceCode string `json:"-" gorm:"size:50;not null;index:idx_activity_events_ns_proj,priority:1"`
	ProjectCode   string `json:"-" gorm:"size:50;not null;index:idx_activity_events_ns_proj,priority:2"`

	Resource ActivityResource `json:"resource" gorm:"size:20;not null"`
	Action   ActivityAction   `json:"action" gorm:"size:20;not null"`

	// Actor is a snapshot: the username for a user session, the token name for an
	// API token. It is deliberately not a foreign key, so the trail stays readable
	// after the user is deleted or renamed. UserID is set to NULL by the database
	// when the user is deleted.
	UserID   *int64            `json:"userID" gorm:"index:idx_activity_events_user"`
	Actor    string            `json:"actor" gorm:"size:300;not null"`
	AuthType appTypes.AuthType `json:"authType" gorm:"size:20"`

	// ResourceID is the redirect or page id when the event targets a single entry,
	// nil for aggregated events (import, project rollback, publish).
	ResourceID *int64       `json:"resourceID" gorm:"index:idx_activity_events_resource"`
	Data       ActivityData `json:"data" gorm:"type:longtext"`

	OccurredAt time.Time `json:"occurredAt" gorm:"type:timestamp;not null"`
}

type ActivityEventList = commonTypes.PaginatedResult[ActivityEvent]
