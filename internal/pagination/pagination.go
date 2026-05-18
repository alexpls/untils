package pagination

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

type Pagination struct {
	PageSize    int
	CurrentPage int
	HasMore     bool
}

func New(pageSize, currentPage int) Pagination {
	return Pagination{PageSize: pageSize, CurrentPage: currentPage}
}

func NewOneBased(pageSize, currentPage int) Pagination {
	return New(pageSize, currentPage-1)
}

func PaginationFromAPIRequest(r *http.Request, defaultPageSize, maxPageSize int) (Pagination, error) {
	currentPage := 1
	pageSize := defaultPageSize

	if pageValue := r.URL.Query().Get("page"); pageValue != "" {
		page, err := strconv.Atoi(pageValue)
		if err != nil || page < 1 {
			return Pagination{}, fmt.Errorf("page must be a positive integer")
		}
		currentPage = page
	}

	if perPageValue := r.URL.Query().Get("per_page"); perPageValue != "" {
		perPage, err := strconv.Atoi(perPageValue)
		if err != nil || perPage < 1 || perPage > maxPageSize {
			return Pagination{}, fmt.Errorf("per_page must be between 1 and %d", maxPageSize)
		}
		pageSize = perPage
	}

	return NewOneBased(pageSize, currentPage), nil
}

func PaginationFromRequest(r *http.Request, pageSize int) Pagination {
	p := New(pageSize, 0)
	if pg := r.URL.Query().Get("page"); pg != "" {
		num, err := strconv.Atoi(pg)
		if err == nil {
			p.CurrentPage = num
		}
	}
	return p
}

func (p Pagination) NextPageParams() url.Values {
	u := url.Values{}
	u.Set("page", strconv.Itoa(p.NextPage()))
	return u
}

func (p Pagination) PrevPageParams() url.Values {
	u := url.Values{}
	u.Set("page", strconv.Itoa(p.PrevPage()))
	return u
}

func (p Pagination) CurrentPageParams() url.Values {
	u := url.Values{}
	u.Set("page", strconv.Itoa(p.CurrentPage))
	return u
}

func (p Pagination) CurrentPageOneBased() int {
	return p.CurrentPage + 1
}

func (p Pagination) Offset() int {
	return p.PageSize * p.CurrentPage
}

func (p Pagination) Offset64() int64 {
	return int64(p.Offset())
}

func (p Pagination) HasNext() bool {
	return p.HasMore
}

func (p Pagination) HasPrev() bool {
	return p.CurrentPage > 0
}

func (p Pagination) NextPage() int {
	if p.HasNext() {
		return p.CurrentPage + 1
	} else {
		return p.CurrentPage
	}
}

func (p Pagination) PrevPage() int {
	if p.HasPrev() {
		return p.CurrentPage - 1
	} else {
		return p.CurrentPage
	}
}

func (p Pagination) PageSizeWithPeek() int {
	return p.PageSize + 1
}

func Peek[T any](items []T, p Pagination) ([]T, Pagination) {
	// TODO: is Peek the best name for this method? methinks no
	if len(items) == p.PageSizeWithPeek() {
		items = items[:p.PageSize]
		p.HasMore = true
	}

	return items, p
}
