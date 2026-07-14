package apperror

import (
	"errors"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
)

type Error struct {
	StatusCode int
	Code       string
	Message    string
	cause      error
}

func New(statusCode int, code, message string, cause error) *Error {
	return &Error{
		StatusCode: statusCode,
		Code:       code,
		Message:    message,
		cause:      cause,
	}
}

func (e *Error) Error() string {
	if e.cause != nil {
		return e.Message + ": " + e.cause.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.cause
}

// Write converts logic-layer errors to stable JSON responses. Unknown errors
// are intentionally hidden from callers.
func Write(w http.ResponseWriter, err error) {
	var appErr *Error
	if errors.As(err, &appErr) {
		httpx.WriteJson(w, appErr.StatusCode, map[string]string{
			"code":    appErr.Code,
			"message": appErr.Message,
		})
		return
	}

	httpx.WriteJson(w, http.StatusInternalServerError, map[string]string{
		"code":    "INTERNAL_ERROR",
		"message": "服务内部错误",
	})
}
