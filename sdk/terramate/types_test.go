package terramate

import "testing"

func TestPaginatedResult_HasNextPage(t *testing.T) {
	tests := []struct {
		name     string
		result   PaginatedResult
		expected bool
	}{
		{
			name:     "has next page",
			result:   PaginatedResult{Total: 100, Page: 1, PerPage: 10},
			expected: true,
		},
		{
			name:     "last page",
			result:   PaginatedResult{Total: 100, Page: 10, PerPage: 10},
			expected: false,
		},
		{
			name:     "partial last page",
			result:   PaginatedResult{Total: 95, Page: 9, PerPage: 10},
			expected: true,
		},
		{
			name:     "partial last page - on last",
			result:   PaginatedResult{Total: 95, Page: 10, PerPage: 10},
			expected: false,
		},
		{
			name:     "zero per page",
			result:   PaginatedResult{Total: 100, Page: 1, PerPage: 0},
			expected: false,
		},
		{
			name:     "empty result",
			result:   PaginatedResult{Total: 0, Page: 1, PerPage: 10},
			expected: false,
		},
		{
			name:     "zero page",
			result:   PaginatedResult{Total: 100, Page: 0, PerPage: 10},
			expected: false,
		},
		{
			name:     "negative page",
			result:   PaginatedResult{Total: 100, Page: -1, PerPage: 10},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.HasNextPage()
			if got != tt.expected {
				t.Errorf("HasNextPage() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPaginatedResult_HasPrevPage(t *testing.T) {
	tests := []struct {
		name     string
		result   PaginatedResult
		expected bool
	}{
		{
			name:     "first page",
			result:   PaginatedResult{Total: 100, Page: 1, PerPage: 10},
			expected: false,
		},
		{
			name:     "second page",
			result:   PaginatedResult{Total: 100, Page: 2, PerPage: 10},
			expected: true,
		},
		{
			name:     "last page",
			result:   PaginatedResult{Total: 100, Page: 10, PerPage: 10},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.HasPrevPage()
			if got != tt.expected {
				t.Errorf("HasPrevPage() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPaginatedResult_TotalPages(t *testing.T) {
	tests := []struct {
		name     string
		result   PaginatedResult
		expected int
	}{
		{
			name:     "exact pages",
			result:   PaginatedResult{Total: 100, Page: 1, PerPage: 10},
			expected: 10,
		},
		{
			name:     "partial last page",
			result:   PaginatedResult{Total: 95, Page: 1, PerPage: 10},
			expected: 10,
		},
		{
			name:     "single page",
			result:   PaginatedResult{Total: 5, Page: 1, PerPage: 10},
			expected: 1,
		},
		{
			name:     "empty result",
			result:   PaginatedResult{Total: 0, Page: 1, PerPage: 10},
			expected: 0,
		},
		{
			name:     "zero per page",
			result:   PaginatedResult{Total: 100, Page: 1, PerPage: 0},
			expected: 0,
		},
		{
			name:     "one item per page",
			result:   PaginatedResult{Total: 100, Page: 1, PerPage: 1},
			expected: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.TotalPages()
			if got != tt.expected {
				t.Errorf("TotalPages() = %v, want %v", got, tt.expected)
			}
		})
	}
}
