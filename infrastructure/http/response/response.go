package response

// Response is the unified response envelope for every API endpoint.
type Response[T any] struct {
	Code    int                 `json:"code"`
	Message string              `json:"message"`
	Data    T                   `json:"data,omitempty"`
	Errors  map[string][]string `json:"errors,omitempty"`
}

func (r *Response[T]) Error() string {
	return r.Message
}
