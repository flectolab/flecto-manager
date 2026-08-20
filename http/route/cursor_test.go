package route

import (
	"encoding/base64"
	"testing"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
	"github.com/flectolab/flecto-manager/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeDecodeCursor(t *testing.T) {
	tests := []struct {
		name   string
		cursor Cursor
	}{
		{
			name:   "first page",
			cursor: Cursor{AfterID: 500, Total: 700000, Delivered: 500},
		},
		{
			name:   "zero values",
			cursor: Cursor{},
		},
		{
			name:   "large ids",
			cursor: Cursor{AfterID: 9007199254740993, Total: 1, Delivered: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := EncodeCursor(tt.cursor)
			require.NoError(t, err)
			// The cursor travels in a query string, so it must survive it unescaped.
			assert.NotContains(t, encoded, "+")
			assert.NotContains(t, encoded, "/")
			assert.NotContains(t, encoded, "=")

			decoded, err := DecodeCursor(encoded)
			require.NoError(t, err)
			assert.Equal(t, tt.cursor, decoded)
		})
	}
}

func TestNewListPage(t *testing.T) {
	tests := []struct {
		name       string
		pagination *commonTypes.PaginationInput
		cursor     *Cursor
		total      int64
		count      int
		lastID     int64
		want       ListPage
		wantNext   *Cursor
	}{
		{
			name:       "offset mode, full page hands out a cursor",
			pagination: &commonTypes.PaginationInput{Limit: types.Ptr(3), Offset: types.Ptr(0)},
			total:      10,
			count:      3,
			lastID:     42,
			want:       ListPage{Total: 10, Offset: 0, Limit: 3},
			wantNext:   &Cursor{AfterID: 42, Total: 10, Delivered: 3},
		},
		{
			name:       "offset mode keeps reporting the offset it was given",
			pagination: &commonTypes.PaginationInput{Limit: types.Ptr(3), Offset: types.Ptr(6)},
			total:      10,
			count:      3,
			lastID:     42,
			want:       ListPage{Total: 10, Offset: 6, Limit: 3},
			wantNext:   &Cursor{AfterID: 42, Total: 10, Delivered: 9},
		},
		{
			name:       "short page ends the walk",
			pagination: &commonTypes.PaginationInput{Limit: types.Ptr(3), Offset: types.Ptr(0)},
			total:      2,
			count:      2,
			lastID:     42,
			want:       ListPage{Total: 2, Offset: 0, Limit: 3},
		},
		{
			name:       "cursor mode reports the total and position it carries",
			pagination: &commonTypes.PaginationInput{Limit: types.Ptr(3)},
			cursor:     &Cursor{AfterID: 42, Total: 10, Delivered: 3},
			total:      0,
			count:      3,
			lastID:     77,
			want:       ListPage{Total: 10, Offset: 3, Limit: 3},
			wantNext:   &Cursor{AfterID: 77, Total: 10, Delivered: 6},
		},
		{
			name:       "empty page ends the walk",
			pagination: &commonTypes.PaginationInput{Limit: types.Ptr(3)},
			cursor:     &Cursor{AfterID: 42, Total: 10, Delivered: 10},
			count:      0,
			want:       ListPage{Total: 10, Offset: 10, Limit: 3},
		},
		{
			name:       "a zero limit hands out no cursor",
			pagination: &commonTypes.PaginationInput{Limit: types.Ptr(0)},
			total:      10,
			count:      0,
			want:       ListPage{Total: 10, Offset: 0, Limit: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewListPage(tt.pagination, tt.cursor, tt.total, tt.count, tt.lastID)
			require.NoError(t, err)

			assert.Equal(t, tt.want.Total, got.Total)
			assert.Equal(t, tt.want.Offset, got.Offset)
			assert.Equal(t, tt.want.Limit, got.Limit)

			if tt.wantNext == nil {
				assert.Empty(t, got.Next)
				return
			}
			require.NotEmpty(t, got.Next)
			next, err := DecodeCursor(got.Next)
			require.NoError(t, err)
			assert.Equal(t, *tt.wantNext, next)
		})
	}
}

func TestDecodeCursor(t *testing.T) {
	tests := []struct {
		name    string
		encoded string
		want    Cursor
		wantErr string
	}{
		{
			name:    "valid",
			encoded: base64.RawURLEncoding.EncodeToString([]byte(`{"id":42,"total":10,"delivered":5}`)),
			want:    Cursor{AfterID: 42, Total: 10, Delivered: 5},
		},
		{
			name:    "unknown fields are ignored",
			encoded: base64.RawURLEncoding.EncodeToString([]byte(`{"id":42,"future":"value"}`)),
			want:    Cursor{AfterID: 42},
		},
		{
			name:    "not base64",
			encoded: "not base64 at all",
			wantErr: "malformed cursor",
		},
		{
			name:    "not json",
			encoded: base64.RawURLEncoding.EncodeToString([]byte(`nonsense`)),
			wantErr: "malformed cursor",
		},
		{
			name:    "negative id",
			encoded: base64.RawURLEncoding.EncodeToString([]byte(`{"id":-1}`)),
			wantErr: "malformed cursor: negative values",
		},
		{
			name:    "negative total",
			encoded: base64.RawURLEncoding.EncodeToString([]byte(`{"id":1,"total":-1}`)),
			wantErr: "malformed cursor: negative values",
		},
		{
			name:    "negative delivered",
			encoded: base64.RawURLEncoding.EncodeToString([]byte(`{"id":1,"delivered":-1}`)),
			wantErr: "malformed cursor: negative values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeCursor(tt.encoded)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Equal(t, Cursor{}, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
