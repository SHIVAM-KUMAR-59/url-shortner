package apperrors

import (
	"errors"
	"net/http"
)

var (
	ErrURLNotFound         = errors.New("url not found")
	ErrURLExpired          = errors.New("url has expired")
	ErrURLInactive         = errors.New("url is inactive")
	ErrInvalidLongURL      = errors.New("invalid or missing long url")
	ErrInternal            = errors.New("internal server error")
	ErrClockMovedBackwards = errors.New("idgen: system clock moved backwards")
	ErrInvalidNodeID       = errors.New("idgen: node id out of range")
	ErrCacheMiss           = errors.New("cache miss") // Don't add this to GetStatusCode as client will never see this
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

	case errors.Is(err, ErrInvalidNodeID):
		return http.StatusInternalServerError

	case errors.Is(err, ErrInternal):
		return http.StatusInternalServerError

	default:
		return http.StatusInternalServerError
	}

}
