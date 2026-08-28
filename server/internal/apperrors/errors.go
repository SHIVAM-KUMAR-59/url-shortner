package apperrors

import (
	"errors"
	"fmt"
	"net/http"
)

type StatusCode int

var ErrCacheMiss = errors.New("Cache miss")

const (
	StatusOK        StatusCode = http.StatusOK
	StatusCreated   StatusCode = http.StatusCreated
	StatusNoContent StatusCode = http.StatusNoContent

	StatusBadRequest          StatusCode = http.StatusBadRequest
	StatusUnauthorized        StatusCode = http.StatusUnauthorized
	StatusForbidden           StatusCode = http.StatusForbidden
	StatusNotFound            StatusCode = http.StatusNotFound
	StatusConflict            StatusCode = http.StatusConflict
	StatusUnprocessableEntity StatusCode = http.StatusUnprocessableEntity
	StatusTooManyRequests     StatusCode = http.StatusTooManyRequests

	StatusInternalServerError StatusCode = http.StatusInternalServerError
	StatusBadGateway          StatusCode = http.StatusBadGateway
	StatusServiceUnavailable  StatusCode = http.StatusServiceUnavailable

	StatusGone StatusCode = http.StatusGone
)

type ErrorCode string

const (
	ErrorCodeBadRequest      ErrorCode = "BAD_REQUEST"
	ErrorCodeValidationError ErrorCode = "VALIDATION_ERROR"

	ErrorCodeUnauthorized ErrorCode = "UNAUTHORIZED"
	ErrorCodeForbidden    ErrorCode = "FORBIDDEN"

	ErrorCodeNotFound ErrorCode = "NOT_FOUND"
	ErrorCodeConflict ErrorCode = "CONFLICT"

	ErrorCodeRateLimited ErrorCode = "RATE_LIMITED"

	ErrorCodeDatabaseError        ErrorCode = "DATABASE_ERROR"
	ErrorCodeExternalServiceError ErrorCode = "EXTERNAL_SERVICE_ERROR"

	ErrorCodeInternalServerError ErrorCode = "INTERNAL_SERVER_ERROR"

	ErrorCodeExpired  ErrorCode = "EXPIRED"
	ErrorCodeInactive ErrorCode = "INACTIVE"
)

type AppError struct {
	Message    string
	StatusCode StatusCode
	ErrorCode  ErrorCode
	Err        error
}

func (e *AppError) Error() string {
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func New(
	message string,
	statusCode StatusCode,
	errorCode ErrorCode,
) *AppError {
	return &AppError{
		Message:    message,
		StatusCode: statusCode,
		ErrorCode:  errorCode,
	}
}

func Wrap(
	err error,
	message string,
	statusCode StatusCode,
	errorCode ErrorCode,
) *AppError {
	return &AppError{
		Message:    message,
		StatusCode: statusCode,
		ErrorCode:  errorCode,
		Err:        err,
	}
}

func BadRequestError(message string) *AppError {
	return New(
		message,
		StatusBadRequest,
		ErrorCodeBadRequest,
	)
}

func ValidationError(message string) *AppError {
	return New(
		message,
		StatusBadRequest,
		ErrorCodeValidationError,
	)
}

func UnauthorizedError(message ...string) *AppError {
	msg := "Unauthorized"
	if len(message) > 0 {
		msg = message[0]
	}

	return New(
		msg,
		StatusUnauthorized,
		ErrorCodeUnauthorized,
	)

}

func ForbiddenError(message ...string) *AppError {
	msg := "Forbidden"
	if len(message) > 0 {
		msg = message[0]
	}

	return New(
		msg,
		StatusForbidden,
		ErrorCodeForbidden,
	)

}

func NotFoundError(resource string, id ...any) *AppError {
	message := fmt.Sprintf("%s not found", resource)

	if len(id) > 0 {
		message = fmt.Sprintf("%s not found: %v", resource, id[0])
	}

	return New(
		message,
		StatusNotFound,
		ErrorCodeNotFound,
	)

}

func ConflictError(message string) *AppError {
	return New(
		message,
		StatusConflict,
		ErrorCodeConflict,
	)
}

func TooManyRequestsError(message string) *AppError {
	return New(
		message,
		StatusTooManyRequests,
		ErrorCodeRateLimited,
	)
}

func InternalServerError(message ...string) *AppError {
	msg := "Internal Server Error"
	if len(message) > 0 {
		msg = message[0]
	}

	return New(
		msg,
		StatusInternalServerError,
		ErrorCodeInternalServerError,
	)

}

func DatabaseError(err error) *AppError {
	return Wrap(
		err,
		"database operation failed",
		StatusInternalServerError,
		ErrorCodeDatabaseError,
	)
}

func ExternalServiceError(message string, err ...error) *AppError {
	var wrappedErr error

	if len(err) > 0 {
		wrappedErr = err[0]
	}

	return &AppError{
		Message:    message,
		StatusCode: StatusServiceUnavailable,
		ErrorCode:  ErrorCodeExternalServiceError,
		Err:        wrappedErr,
	}

}

func GetStatusCode(err error) int {
	var appErr *AppError

	if errors.As(err, &appErr) {
		return int(appErr.StatusCode)
	}

	return http.StatusInternalServerError

}

func GetErrorCode(err error) ErrorCode {
	var appErr *AppError

	if errors.As(err, &appErr) {
		return appErr.ErrorCode
	}

	return ErrorCodeInternalServerError

}

func ExpiredError(message string) *AppError {
	return New(
		message,
		StatusGone,
		ErrorCodeExpired,
	)
}

func InactiveError(message string) *AppError {
	return New(
		message,
		StatusGone,
		ErrorCodeInactive,
	)
}
