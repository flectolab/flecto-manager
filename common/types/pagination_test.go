package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func intPtr(i int) *int {
	return &i
}

func TestPaginationInput_GetLimit(t *testing.T) {
	tests := []struct {
		name  string
		input *PaginationInput
		want  int
	}{
		{
			name:  "nil receiver returns default",
			input: nil,
			want:  DefaultLimit,
		},
		{
			name:  "nil Limit returns default",
			input: &PaginationInput{Limit: nil},
			want:  DefaultLimit,
		},
		{
			name:  "custom Limit",
			input: &PaginationInput{Limit: intPtr(50)},
			want:  50,
		},
		{
			name:  "zero Limit",
			input: &PaginationInput{Limit: intPtr(0)},
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.input.GetLimit()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPaginationInput_GetOffset(t *testing.T) {
	tests := []struct {
		name  string
		input *PaginationInput
		want  int
	}{
		{
			name:  "nil receiver returns default",
			input: nil,
			want:  DefaultOffset,
		},
		{
			name:  "nil Offset returns default",
			input: &PaginationInput{Offset: nil},
			want:  DefaultOffset,
		},
		{
			name:  "custom Offset",
			input: &PaginationInput{Offset: intPtr(100)},
			want:  100,
		},
		{
			name:  "zero Offset",
			input: &PaginationInput{Offset: intPtr(0)},
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.input.GetOffset()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPaginatedResult_HasMore(t *testing.T) {
	tests := []struct {
		name   string
		result PaginatedResult[int]
		want   bool
	}{
		{
			name: "has more items",
			result: PaginatedResult[int]{
				Items:  []int{1, 2, 3},
				Total:  10,
				Offset: 0,
			},
			want: true,
		},
		{
			name: "no more items - exact end",
			result: PaginatedResult[int]{
				Items:  []int{1, 2, 3},
				Total:  3,
				Offset: 0,
			},
			want: false,
		},
		{
			name: "no more items - last page",
			result: PaginatedResult[int]{
				Items:  []int{9, 10},
				Total:  10,
				Offset: 8,
			},
			want: false,
		},
		{
			name: "empty items with total",
			result: PaginatedResult[int]{
				Items:  []int{},
				Total:  10,
				Offset: 0,
			},
			want: true,
		},
		{
			name: "empty items no total",
			result: PaginatedResult[int]{
				Items:  []int{},
				Total:  0,
				Offset: 0,
			},
			want: false,
		},
		{
			name: "middle of pagination",
			result: PaginatedResult[int]{
				Items:  []int{1, 2, 3, 4, 5},
				Total:  100,
				Offset: 20,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.HasMore()
			assert.Equal(t, tt.want, got)
		})
	}
}
