package responder

import (
	"fmt"
	"math"

	"CredChain_Golang/domain"
	"github.com/gin-gonic/gin"
)

// PaginatedData structures the paginated response identically to the alphabetical JS format requirement
type PaginatedData[T any] struct {
	FirstPageURL *string `json:"first_page_url"`
	From         int     `json:"from"`
	Items        []T     `json:"items"`
	LastPage     int     `json:"last_page"`
	LastPageURL  *string `json:"last_page_url"`
	Limit        int     `json:"limit"`
	NextPageURL  *string `json:"next_page_url"`
	Page         int     `json:"page"`
	PrevPageURL  *string `json:"prev_page_url"`
	To           int     `json:"to"`
	Total        int     `json:"total"`
}

// BuildPaginated constructs a PaginatedData struct based on items, total, and original query.
func BuildPaginated[T any](c *gin.Context, items []T, total int, query domain.Query) PaginatedData[T] {
	if items == nil {
		items = make([]T, 0)
	}

	page := query.Page
	if page < 1 {
		page = 1
	}
	limit := query.Limit
	if limit < 1 {
		limit = 10
	}

	lastPage := int(math.Ceil(float64(total) / float64(limit)))
	if lastPage < 1 {
		lastPage = 1
	}

	from := (page - 1) * limit + 1
	to := page * limit
	if total == 0 {
		from = 0
		to = 0
	} else if to > total {
		to = total
	}

	baseUrl := fmt.Sprintf("%s://%s%s", "http", c.Request.Host, c.Request.URL.Path)

	buildURL := func(targetPage int) *string {
		q := c.Request.URL.Query()
		q.Set("page", fmt.Sprintf("%d", targetPage))
		q.Set("limit", fmt.Sprintf("%d", limit))
		urlStr := fmt.Sprintf("%s?%s", baseUrl, q.Encode())
		return &urlStr
	}

	firstPageUrl := buildURL(1)
	lastPageUrl := buildURL(lastPage)
	
	var prevPageUrl *string
	if page > 1 {
		prevPageUrl = buildURL(page - 1)
	}
	
	var nextPageUrl *string
	if page < lastPage {
		nextPageUrl = buildURL(page + 1)
	}

	return PaginatedData[T]{
		FirstPageURL: firstPageUrl,
		From:         from,
		Items:        items,
		LastPage:     lastPage,
		LastPageURL:  lastPageUrl,
		Limit:        limit,
		NextPageURL:  nextPageUrl,
		Page:         page,
		PrevPageURL:  prevPageUrl,
		To:           to,
		Total:        total,
	}
}

// SendPaginated wraps PaginatedData in standard Response
func SendPaginated[T any](c *gin.Context, code int, items []T, total int, query domain.Query) {
	data := BuildPaginated(c, items, total, query)
	Send(c, code, data)
}
