package domain

// Query defines standard query parameters for listing items
type Query struct {
	Page     int               `json:"page"`
	Limit    int               `json:"limit"`
	Search   string            `json:"search"`
	Sorts    []string          `json:"sorts"`    // Example: ["created_at", "-name"]
	Filters  map[string]string `json:"filters"`  // Example: {"role": "issuer"}
	Includes []string          `json:"includes"` // For SQL Joins
	Groups   []string          `json:"groups"`   // For GROUP BY clauses
}

// DefaultQuery returns a query with default sensible limits
func DefaultQuery() Query {
	return Query{
		Page:  1,
		Limit: 10,
	}
}

// Offset returns the SQL offset for the current page and limit
func (q *Query) Offset() int {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 {
		q.Limit = 10
	}
	return (q.Page - 1) * q.Limit
}
