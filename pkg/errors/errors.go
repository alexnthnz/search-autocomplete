package errors

import (
	"fmt"
	"net/http"
)

// ErrorCode represents different types of errors
type ErrorCode string

const (
	ErrCodeValidation   ErrorCode = "VALIDATION_ERROR"
	ErrCodeNotFound     ErrorCode = "NOT_FOUND"
	ErrCodeRateLimit    ErrorCode = "RATE_LIMIT_EXCEEDED"
	ErrCodeInternal     ErrorCode = "INTERNAL_ERROR"
	ErrCodeUnauthorized ErrorCode = "UNAUTHORIZED"
	ErrCodeBadRequest   ErrorCode = "BAD_REQUEST"
	ErrCodeCacheFailure ErrorCode = "CACHE_FAILURE"
	ErrCodeTrieFailure  ErrorCode = "TRIE_FAILURE"
	ErrCodeTimeout      ErrorCode = "TIMEOUT"
)

// APIError represents a structured API error
type APIError struct {
	Code       ErrorCode `json:"code"`
	Message    string    `json:"message"`
	Details    string    `json:"details,omitempty"`
	HTTPStatus int       `json:"-"`
	Cause      error     `json:"-"`
}

// Error implements the error interface
func (e *APIError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *APIError) Unwrap() error {
	return e.Cause
}

// NewValidationError creates a validation error
func NewValidationError(message, details string) *APIError {
	return &APIError{
		Code:       ErrCodeValidation,
		Message:    message,
		Details:    details,
		HTTPStatus: http.StatusBadRequest,
	}
}

// NewNotFoundError creates a not found error
func NewNotFoundError(resource string) *APIError {
	return &APIError{
		Code:       ErrCodeNotFound,
		Message:    fmt.Sprintf("%s not found", resource),
		HTTPStatus: http.StatusNotFound,
	}
}

// NewRateLimitError creates a rate limit error
func NewRateLimitError() *APIError {
	return &APIError{
		Code:       ErrCodeRateLimit,
		Message:    "Rate limit exceeded",
		Details:    "Please slow down your requests",
		HTTPStatus: http.StatusTooManyRequests,
	}
}

// NewInternalError creates an internal server error
func NewInternalError(message string, cause error) *APIError {
	return &APIError{
		Code:       ErrCodeInternal,
		Message:    message,
		HTTPStatus: http.StatusInternalServerError,
		Cause:      cause,
	}
}

// NewUnauthorizedError creates an unauthorized error
func NewUnauthorizedError(message string) *APIError {
	return &APIError{
		Code:       ErrCodeUnauthorized,
		Message:    message,
		HTTPStatus: http.StatusUnauthorized,
	}
}

// NewCacheError creates a cache-related error
func NewCacheError(operation string, cause error) *APIError {
	return &APIError{
		Code:       ErrCodeCacheFailure,
		Message:    fmt.Sprintf("Cache %s failed", operation),
		HTTPStatus: http.StatusInternalServerError,
		Cause:      cause,
	}
}

// NewTrieError creates a trie-related error
func NewTrieError(operation string, cause error) *APIError {
	return &APIError{
		Code:       ErrCodeTrieFailure,
		Message:    fmt.Sprintf("Trie %s failed", operation),
		HTTPStatus: http.StatusInternalServerError,
		Cause:      cause,
	}
}

// NewTimeoutError creates a timeout error
func NewTimeoutError(operation string) *APIError {
	return &APIError{
		Code:       ErrCodeTimeout,
		Message:    fmt.Sprintf("Operation '%s' timed out", operation),
		HTTPStatus: http.StatusRequestTimeout,
	}
}

// WrapError wraps an existing error with additional context
func WrapError(err error, code ErrorCode, message string) *APIError {
	return &APIError{
		Code:       code,
		Message:    message,
		HTTPStatus: http.StatusInternalServerError,
		Cause:      err,
	}
}

// IsAPIError checks if an error is an APIError
func IsAPIError(err error) bool {
	_, ok := err.(*APIError)
	return ok
}

// GetHTTPStatus returns the HTTP status code for an error
func GetHTTPStatus(err error) int {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.HTTPStatus
	}
	return http.StatusInternalServerError
}
