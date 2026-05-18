package pagination

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewOneBased(t *testing.T) {
	t.Parallel()

	p := NewOneBased(50, 3)

	require.Equal(t, 50, p.PageSize)
	require.Equal(t, 2, p.CurrentPage)
	require.Equal(t, 100, p.Offset())
	require.Equal(t, int64(100), p.Offset64())
}

func TestPaginationFromAPIRequest(t *testing.T) {
	t.Parallel()

	r := httptest.NewRequest("GET", "/api/results.list?page=3&per_page=25", nil)

	p, err := PaginationFromAPIRequest(r, 50, 100)

	require.NoError(t, err)
	require.Equal(t, 25, p.PageSize)
	require.Equal(t, 2, p.CurrentPage)
	require.Equal(t, 3, p.CurrentPageOneBased())
	require.Equal(t, 50, p.Offset())
}

func TestPaginationFromAPIRequestAcceptsLargePage(t *testing.T) {
	t.Parallel()

	r := httptest.NewRequest("GET", "/api/results.list?page=2147483648&per_page=1", nil)

	p, err := PaginationFromAPIRequest(r, 50, 100)

	require.NoError(t, err)
	require.Equal(t, 1, p.PageSize)
	require.Equal(t, 2147483647, p.CurrentPage)
	require.Equal(t, 2147483648, p.CurrentPageOneBased())
}

func TestPaginationFromAPIRequestDefaults(t *testing.T) {
	t.Parallel()

	r := httptest.NewRequest("GET", "/api/results.list", nil)

	p, err := PaginationFromAPIRequest(r, 50, 100)

	require.NoError(t, err)
	require.Equal(t, 50, p.PageSize)
	require.Equal(t, 0, p.CurrentPage)
}

func TestPaginationFromAPIRequestRejectsInvalidPage(t *testing.T) {
	t.Parallel()

	r := httptest.NewRequest("GET", "/api/results.list?page=0", nil)

	_, err := PaginationFromAPIRequest(r, 50, 100)

	require.EqualError(t, err, "page must be a positive integer")
}

func TestPaginationFromAPIRequestRejectsInvalidPerPage(t *testing.T) {
	t.Parallel()

	r := httptest.NewRequest("GET", "/api/results.list?per_page=101", nil)

	_, err := PaginationFromAPIRequest(r, 50, 100)

	require.EqualError(t, err, "per_page must be between 1 and 100")
}

func TestPeekTrimsExtraItemAndMarksHasMore(t *testing.T) {
	t.Parallel()

	items, p := Peek([]int{1, 2, 3}, New(2, 0))

	require.Equal(t, []int{1, 2}, items)
	require.True(t, p.HasMore)
}

func TestPeekLeavesShortPageUnchanged(t *testing.T) {
	t.Parallel()

	items, p := Peek([]int{1}, New(2, 0))

	require.Equal(t, []int{1}, items)
	require.False(t, p.HasMore)
}
