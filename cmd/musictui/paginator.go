package main

import "fmt"

// PaginationState tracks pagination and search for a browse page.
type PaginationState struct {
	Page       int
	PageSize   int
	TotalPages int
	Total      int
	Search     string
}

// NewPaginationState creates a new pagination state with the given page size.
func NewPaginationState(pageSize int) *PaginationState {
	return &PaginationState{Page: 1, PageSize: pageSize}
}

// NextPage advances to the next page. Returns true if the page changed.
func (p *PaginationState) NextPage() bool {
	if p.Page < p.TotalPages {
		p.Page++
		return true
	}
	return false
}

// PrevPage goes to the previous page. Returns true if the page changed.
func (p *PaginationState) PrevPage() bool {
	if p.Page > 1 {
		p.Page--
		return true
	}
	return false
}

// SetSearch sets the search query and resets to page 1.
func (p *PaginationState) SetSearch(s string) {
	p.Search = s
	p.Page = 1
}

// ClearSearch clears the search query and resets to page 1.
func (p *PaginationState) ClearSearch() {
	p.Search = ""
	p.Page = 1
}

// UpdateFromResponse updates pagination state from an API response.
func (p *PaginationState) UpdateFromResponse(page, totalPages, total int) {
	p.Page = page
	p.TotalPages = totalPages
	p.Total = total
}

// StatusText returns a human-readable pagination status string.
func (p *PaginationState) StatusText() string {
	if p.TotalPages <= 1 {
		text := fmt.Sprintf("%d items", p.Total)
		if p.Search != "" {
			text += fmt.Sprintf(" matching \"%s\"", p.Search)
		}
		return text
	}
	text := fmt.Sprintf("Page %d/%d (%d items)", p.Page, p.TotalPages, p.Total)
	if p.Search != "" {
		text += fmt.Sprintf(" matching \"%s\"", p.Search)
	}
	return text
}
