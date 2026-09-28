package dto

// PaginationMeta is the pagination bookkeeping alongside a page of results:
// the total number of records matching a filter (before pagination) plus the
// page/limit that produced this page, so a frontend can render page
// controls. Not scoped to catalog specifically - any paginated list result
// can reuse it.
type PaginationMeta struct {
	Total int `json:"total"`
	Page  int `json:"page"`
	Limit int `json:"limit"`
}
