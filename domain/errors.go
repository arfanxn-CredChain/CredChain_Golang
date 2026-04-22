package domain

// Error represents a business logic error in the domain layer.
type Error struct {
	Code     int
	Err      error
	Metadata map[string]any
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Err == nil {
		return "operation failed"
	}
	return e.Err.Error()
}

// Unwrap returns the underlying error for errors.Is/As.
func (e *Error) Unwrap() error {
	return e.Err
}

// ErrorOption configures an Error
type ErrorOption func(*Error)

// WithError sets the underlying error
func WithError(err error) ErrorOption {
	return func(e *Error) {
		e.Err = err
	}
}

// WithMetadata sets a single metadata key-value pair
func WithMetadata(key string, value any) ErrorOption {
	return func(e *Error) {
		if e.Metadata == nil {
			e.Metadata = make(map[string]any)
		}
		e.Metadata[key] = value
	}
}

// WithMetadataMap sets multiple metadata key-value pairs
func WithMetadataMap(metadata map[string]any) ErrorOption {
	return func(e *Error) {
		if e.Metadata == nil {
			e.Metadata = make(map[string]any)
		}
		for k, v := range metadata {
			e.Metadata[k] = v
		}
	}
}

// NewError creates a new Error with the given code and options
func NewError(code int, opts ...ErrorOption) *Error {
	e := &Error{
		Code: code,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}
