package error

import "net/http"

type AppError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func (e *AppError) Error() string {
	return e.Message
}

func New(message string, code int) *AppError {
	return &AppError{
		Message: message,
		Code:    code,
	}
}
func InternalError() *AppError {
	return New("Internal server error", http.StatusInternalServerError)
}

func ServiceUnavailable(msg string) *AppError {
	if msg == "" {
		msg = "Service temporarily unavailable, please try again"
	}
	return New(msg, http.StatusServiceUnavailable)
}

func BadRequest(msg string) *AppError {
	if msg == "" {
		msg = "Invalid request"
	}
	return New(msg, http.StatusBadRequest)
}

func NotFound(msg string) *AppError {
	if msg == "" {
		msg = "Resource not found"
	}
	return New(msg, http.StatusNotFound)
}

func Conflict(msg string) *AppError {
	if msg == "" {
		msg = "Conflict occurred"
	}
	return New(msg, http.StatusConflict)
}
