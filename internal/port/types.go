package port

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
)

const (
	// CodeVersionConflict signals an optimistic-locking version mismatch.
	CodeVersionConflict = 1004
	// CodeTooManyRequests signals the client exceeded the rate limit.
	CodeTooManyRequests = 1005
	// CodeIdempotencyInFlight signals a request with the same key is still running.
	CodeIdempotencyInFlight = 1006
)

// ErrorCoder is implemented by errors that carry a unified business error code.
// Usecase sentinel errors implement it; port.ResponseError uses it to map errors
// to the unified envelope without leaking internal details to clients.
type ErrorCoder interface {
	error
	ErrorCode() int
}

type codedError struct {
	code int
	msg  string
}

func (e *codedError) Error() string  { return e.msg }
func (e *codedError) ErrorCode() int { return e.code }

// Is lets errors.Is match coded sentinels by code+msg, so repositories can wrap
// them with %w and handlers can still detect them.
func (e *codedError) Is(target error) bool {
	t, ok := target.(*codedError)
	return ok && t.code == e.code && t.msg == e.msg
}

// NewCodedError creates a business error carrying a business code.
// Sentinel errors should be declared as package-level vars.
func NewCodedError(code int, format string, args ...any) error {
	return &codedError{code: code, msg: fmt.Sprintf(format, args...)}
}

// ResponseError maps a repository/usecase error to the unified envelope.
// Errors implementing ErrorCoder map to their business code; everything else
// maps to the internal error code with a generic message (no internals leaked).
func ResponseError(c *gin.Context, err error) {
	var ce ErrorCoder
	if errors.As(err, &ce) {
		Error(c, ce.ErrorCode(), ce.Error())
		return
	}
	ErrorInternal(c, "internal error")
}

// Page is a paginated result for list endpoints. Items holds the page slice
// (concrete slice per use case); it is typed any so swag can render it.
type Page struct {
	Items    any   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

// NewPage builds a Page from a typed slice.
func NewPage[T any](items []T, total int64, p PageParams) Page {
	if items == nil {
		items = []T{}
	}
	return Page{
		Items:    items,
		Total:    total,
		Page:     p.Page,
		PageSize: p.PageSize,
	}
}

// PageParams captures validated pagination from a request.
type PageParams struct {
	Page     int
	PageSize int
}

// PageConfig holds pagination defaults for handlers (usually from config).
type PageConfig struct {
	DefaultSize int
	MaxSize     int
}

// ParsePage clamps page/pageSize into sane bounds.
func ParsePage(page, pageSize, defaultSize, maxSize int) PageParams {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultSize
	}
	if maxSize > 0 && pageSize > maxSize {
		pageSize = maxSize
	}
	return PageParams{Page: page, PageSize: pageSize}
}

// WithDefaults clamps a page/pageSize against a PageConfig.
func (c PageConfig) WithDefaults(page, pageSize int) PageParams {
	return ParsePage(page, pageSize, c.DefaultSize, c.MaxSize)
}
