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
