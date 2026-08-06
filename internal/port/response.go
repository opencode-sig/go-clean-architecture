// Package port defines shared cross-module contracts: transaction interfaces,
// cross-module service interfaces, and the unified HTTP response envelope.
package port

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response is the unified API envelope. HTTP status is always 200;
// business success/failure is signaled by Code.
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// Business error codes, grouped by segment:
//
//	0      success
//	100x   common errors (params, auth, not found, internal)
//	200x   user module
//	300x   article module
//	400x   comment module
const (
	CodeSuccess      = 0
	CodeBadRequest   = 1001
	CodeUnauthorized = 1002
	CodeNotFound     = 1003
	CodeInternal     = 1999

	CodeUserNotFound = 2001

	CodeArticleNotFound = 3001

	CodeCommentNotFound = 4001
)

// Success writes a 200 response with the given business data.
func Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: "",
		Data:    data,
	})
}

// Error writes a 200 response carrying a business error code and message.
func Error(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}

// ErrorInternal writes a 200 response with the internal error code.
func ErrorInternal(c *gin.Context, message string) {
	Error(c, CodeInternal, message)
}
