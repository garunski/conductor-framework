package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-logr/logr"
)

type ErrorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

func WriteJSONResponse(w http.ResponseWriter, logger logr.Logger, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.Error(err, "failed to encode JSON response")
	}
}

func WriteErrorResponse(w http.ResponseWriter, logger logr.Logger, status int, errorMsg string, message string, details map[string]string) {
	resp := ErrorResponse{
		Error:   errorMsg,
		Message: message,
		Details: details,
	}
	WriteJSONResponse(w, logger, status, resp)
}

func WriteError(w http.ResponseWriter, logger logr.Logger, err error) {
	if err == nil {
		WriteErrorResponse(w, logger, http.StatusInternalServerError, "unknown_error", "An unknown error occurred", nil)
		return
	}

	code := extractErrorCode(err)
	status := httpStatus(err)
	message := err.Error()

	WriteErrorResponse(w, logger, status, code, message, nil)
}

func WriteYAMLResponse(w http.ResponseWriter, logger logr.Logger, data []byte) {
	w.Header().Set("Content-Type", "application/yaml")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		logger.Error(err, "failed to write YAML response")
	}
}

