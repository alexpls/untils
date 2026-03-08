package pagination

import (
	"net/http"
	"net/url"
	"strconv"
)

type Pagination struct {
	PageSize    int
	CurrentPage int
	HasMore     bool
}

func PaginationFromRequest(r *http.Request, pageSize int) Pagination {
	p := Pagination{PageSize: pageSize, CurrentPage: 0}
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

func (p Pagination) Offset() int {
	return p.PageSize * p.CurrentPage
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
