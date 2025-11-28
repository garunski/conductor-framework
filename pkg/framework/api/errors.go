package api

import (
	"errors"
	"net/http"

	apperrors "github.com/garunski/conductor-framework/pkg/framework/errors"
)

func httpStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}

	if errors.Is(err, apperrors.ErrNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, apperrors.ErrInvalid) || errors.Is(err, apperrors.ErrInvalidYAML) ||
		errors.Is(err, apperrors.ErrMissingParameter) || errors.Is(err, apperrors.ErrInvalidParameter) ||
		errors.Is(err, apperrors.ErrInvalidRequest) || errors.Is(err, apperrors.ErrInvalidNamespace) ||
		errors.Is(err, apperrors.ErrInvalidServiceName) {
		return http.StatusBadRequest
	}
	if errors.Is(err, apperrors.ErrStorage) {
		return http.StatusInternalServerError
	}
	if errors.Is(err, apperrors.ErrKubernetes) || errors.Is(err, apperrors.ErrReconciliation) {
		return http.StatusInternalServerError
	}
	if errors.Is(err, apperrors.ErrEventStore) {
		return http.StatusServiceUnavailable
	}

	return http.StatusInternalServerError
}

func extractErrorCode(err error) string {
	if err == nil {
		return "unknown_error"
	}

	if errors.Is(err, apperrors.ErrNotFound) {
		return "not_found"
	}
	if errors.Is(err, apperrors.ErrMissingParameter) {
		return "missing_parameter"
	}
	if errors.Is(err, apperrors.ErrInvalidParameter) {
		return "invalid_parameter"
	}
	if errors.Is(err, apperrors.ErrInvalidRequest) {
		return "invalid_request"
	}
	if errors.Is(err, apperrors.ErrInvalidNamespace) {
		return "invalid_namespace"
	}
	if errors.Is(err, apperrors.ErrInvalidServiceName) {
		return "invalid_service_name"
	}
	if errors.Is(err, apperrors.ErrInvalid) {
		return "validation_error"
	}
	if errors.Is(err, apperrors.ErrInvalidYAML) {
		return "invalid_yaml"
	}
	if errors.Is(err, apperrors.ErrStorage) {
		return "storage_error"
	}
	if errors.Is(err, apperrors.ErrKubernetes) {
		return "kubernetes_error"
	}
	if errors.Is(err, apperrors.ErrReconciliation) {
		return "reconciliation_error"
	}
	if errors.Is(err, apperrors.ErrEventStore) {
		return "event_store_unavailable"
	}

	return "internal_error"
}

