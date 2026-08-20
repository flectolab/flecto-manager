package route

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
)

// Cursor is the position and loop state of a keyset-paginated listing.
//
// It is encoded into an opaque string handed to the client and echoed back
// unchanged, so its shape stays a server implementation detail. It carries more
// than the position on purpose:
//
//   - Total, so the count is computed once on the first page instead of on every
//     page. A full sync of a large project otherwise spends most of its time
//     counting the same rows over and over.
//   - Delivered, so the response can keep reporting Offset and existing clients'
//     HasMore keeps working unchanged.
type Cursor struct {
	// AfterID is the last id of the previous page. The next page starts above it.
	AfterID int64 `json:"id"`
	// Total is the row count measured when the listing started.
	Total int `json:"total"`
	// Delivered is how many rows the previous pages returned.
	Delivered int `json:"delivered"`
}

// EncodeCursor renders a cursor as the opaque string returned in Next.
func EncodeCursor(cursor Cursor) (string, error) {
	payload, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

// DecodeCursor parses a cursor received from a client. A malformed cursor is a
// client error, not a server one: it is reported so the caller can answer 400
// rather than silently falling back to offset pagination, which would restart the
// listing from the top and loop forever.
func DecodeCursor(encoded string) (Cursor, error) {
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return Cursor{}, fmt.Errorf("malformed cursor: %w", err)
	}

	var cursor Cursor
	if err := json.Unmarshal(payload, &cursor); err != nil {
		return Cursor{}, fmt.Errorf("malformed cursor: %w", err)
	}
	if cursor.AfterID < 0 || cursor.Total < 0 || cursor.Delivered < 0 {
		return Cursor{}, fmt.Errorf("malformed cursor: negative values")
	}

	return cursor, nil
}

// ListPage holds the pagination fields of a listing response, shared by every
// endpoint that pages through published resources.
type ListPage struct {
	Total  int
	Offset int
	Limit  int
	Next   string
}

// NewListPage computes the pagination fields of a response. cursor is nil when the
// client paginates by offset, in which case total is the count the query measured;
// otherwise the total and the position come from the cursor, so the response looks the
// same either way and clients relying on Offset and Total keep working.
//
// count is the number of rows in this page and lastID the id of its last row, used to
// build the cursor of the following page.
func NewListPage(pagination *commonTypes.PaginationInput, cursor *Cursor, total int64, count int, lastID int64) (ListPage, error) {
	page := ListPage{
		Total:  int(total),
		Offset: pagination.GetOffset(),
		Limit:  pagination.GetLimit(),
	}
	if cursor != nil {
		page.Total = cursor.Total
		page.Offset = cursor.Delivered
	}

	// A short page is the last one, so no cursor is handed out. A full page gets one
	// even when it happens to be the last: the extra request costs a lookup and avoids
	// having to trust the total to decide when to stop.
	if page.Limit > 0 && count == page.Limit {
		next, err := EncodeCursor(Cursor{
			AfterID:   lastID,
			Total:     page.Total,
			Delivered: page.Offset + count,
		})
		if err != nil {
			return ListPage{}, err
		}
		page.Next = next
	}

	return page, nil
}
