package response

// PaginationMeta holds pagination metadata for list responses.
type PaginationMeta struct {
	Total      int `json:"total"`
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int `json:"total_pages"`
}

// PaginatedResponse is the standard envelope for all list endpoints.
type PaginatedResponse[T any] struct {
	Data       []T            `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

// NewPaginated creates a paginated response from raw data and total count.
func NewPaginated[T any](data []T, total, page, perPage int) PaginatedResponse[T] {
	totalPages := 0
	if perPage > 0 {
		totalPages = (total + perPage - 1) / perPage
	}
	if data == nil {
		data = []T{} // never return null in JSON
	}
	return PaginatedResponse[T]{
		Data: data,
		Pagination: PaginationMeta{
			Total:      total,
			Page:       page,
			PerPage:    perPage,
			TotalPages: totalPages,
		},
	}
}
