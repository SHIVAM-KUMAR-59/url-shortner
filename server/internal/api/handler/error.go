package handler

import (
	"encoding/json"
	"net/http"

	"github.com/SHIVAM-KUMAR-59/url-shortener/internal/apperrors"
)

type errorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func writeError(w http.ResponseWriter, err error) {
	statusCode := apperrors.GetStatusCode(err)
	errorCode := apperrors.GetErrorCode(err)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	_ = json.NewEncoder(w).Encode(errorResponse{
		Error: err.Error(),
		Code:  string(errorCode),
	})

}
