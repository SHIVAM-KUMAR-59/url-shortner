package apperrors

import (
	"errors"
	"net/http"
)

var (
	ErrURLNotFound    = errors.New("url not found")
	ErrURLExpired     = errors.New("url has expired")
	ErrURLInactive    = errors.New("url is inactive")
	ErrInvalidLongURL = errors.New("invalid or missing long url")
	ErrInternal       = errors.New("internal server error")
)

func GetStatusCode(err error) int {
	switch {
	case errors.Is(err, ErrURLNotFound):
		return http.StatusNotFound

	case errors.Is(err, ErrURLExpired):
		return http.StatusGone

	case errors.Is(err, ErrURLInactive):
		return http.StatusGone

	case errors.Is(err, ErrInvalidLongURL):
		return http.StatusBadRequest

	default:
		return http.StatusInternalServerError
	}

}
