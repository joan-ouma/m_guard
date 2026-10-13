package domain

type PaginationMeta struct {
	Total      int    `json:"total"`
	Limit      int    `json:"limit"`
	Cursor     string `json:"cursor,omitempty"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

type PaginatedBreadcrumbsResponse struct {
	Data       []BreadcrumbDTO `json:"data"`
	Pagination PaginationMeta  `json:"pagination"`
}

type PaginatedEventsResponse struct {
	Data       []BikeEventDTO `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}
