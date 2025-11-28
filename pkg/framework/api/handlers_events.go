package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	apperrors "github.com/garunski/conductor-framework/pkg/framework/errors"
)

func hasParam(rctx *chi.Context, key string) bool {
	if rctx == nil {
		return false
	}
	for i, k := range rctx.URLParams.Keys {
		if k == key && i < len(rctx.URLParams.Values) {
			return true
		}
	}
	return false
}

func (h *Handler) ListEvents(w http.ResponseWriter, r *http.Request) {
	if h.eventStore == nil {
		WriteError(w, h.logger, fmt.Errorf("%w: event store not available", apperrors.ErrEventStore))
		return
	}

	filters, err := ParseQueryParams(r)
	if err != nil {
		WriteError(w, h.logger, fmt.Errorf("%w: invalid query parameters: %w", apperrors.ErrInvalid, err))
		return
	}

	eventList, err := h.eventStore.ListEvents(filters)
	if err != nil {
		h.logger.Error(err, "failed to list events")
		WriteError(w, h.logger, err)
		return
	}

	WriteJSONResponse(w, h.logger, http.StatusOK, eventList)
}

func (h *Handler) GetEventsByResource(w http.ResponseWriter, r *http.Request) {

	rctx := chi.RouteContext(r.Context())
	resourceKey := chi.URLParam(r, "resourceKey")

	if resourceKey == "" && (rctx == nil || len(rctx.URLParams.Keys) == 0 || !hasParam(rctx, "resourceKey")) {
		resourceKey = chi.URLParam(r, "*")

		if resourceKey == "" && (rctx == nil || len(rctx.URLParams.Keys) == 0 || !hasParam(rctx, "*")) {
			resourceKey = r.URL.Path
			if len(resourceKey) > 0 && resourceKey[0] == '/' {
				resourceKey = resourceKey[1:]
			}

			if len(resourceKey) >= 11 && resourceKey[:11] == "api/events/" {
				resourceKey = resourceKey[11:]
			} else if resourceKey == "api/events" {
				resourceKey = ""
			}
		}
	}
	if err := ValidateKey(resourceKey); err != nil {
		WriteError(w, h.logger, fmt.Errorf("%w: invalid resourceKey: %w", apperrors.ErrInvalid, err))
		return
	}

	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil || parsedLimit <= 0 {
			WriteError(w, h.logger, fmt.Errorf("%w: limit must be a positive integer", apperrors.ErrInvalid))
			return
		}
		if parsedLimit > 1000 {
			WriteError(w, h.logger, fmt.Errorf("%w: limit cannot exceed 1000", apperrors.ErrInvalid))
			return
		}
		limit = parsedLimit
	}

	if h.eventStore == nil {
		WriteError(w, h.logger, fmt.Errorf("%w: event store not available", apperrors.ErrEventStore))
		return
	}

	eventList, err := h.eventStore.GetEventsByResource(resourceKey, limit)
	if err != nil {
		h.logger.Error(err, "failed to get events by resource", "resource", resourceKey)
		WriteError(w, h.logger, err)
		return
	}

	WriteJSONResponse(w, h.logger, http.StatusOK, eventList)
}

func (h *Handler) GetRecentErrors(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil || parsedLimit <= 0 {
			WriteError(w, h.logger, fmt.Errorf("%w: limit must be a positive integer", apperrors.ErrInvalid))
			return
		}
		if parsedLimit > 1000 {
			WriteError(w, h.logger, fmt.Errorf("%w: limit cannot exceed 1000", apperrors.ErrInvalid))
			return
		}
		limit = parsedLimit
	}

	if h.eventStore == nil {
		WriteError(w, h.logger, fmt.Errorf("%w: event store not available", apperrors.ErrEventStore))
		return
	}

	eventList, err := h.eventStore.GetRecentErrors(limit)
	if err != nil {
		h.logger.Error(err, "failed to get recent errors")
		WriteError(w, h.logger, err)
		return
	}

	WriteJSONResponse(w, h.logger, http.StatusOK, eventList)
}

func (h *Handler) CleanupEvents(w http.ResponseWriter, r *http.Request) {
	beforeStr := r.URL.Query().Get("before")
	if beforeStr == "" {
		WriteError(w, h.logger, fmt.Errorf("%w: before parameter is required", apperrors.ErrMissingParameter))
		return
	}

	before, err := time.Parse(time.RFC3339, beforeStr)
	if err != nil {
		WriteError(w, h.logger, fmt.Errorf("%w: invalid before parameter format (use RFC3339): %w", apperrors.ErrInvalidParameter, err))
		return
	}

	if h.eventStore == nil {
		WriteError(w, h.logger, fmt.Errorf("%w: event store not available", apperrors.ErrEventStore))
		return
	}

	if err := h.eventStore.CleanupOldEvents(before); err != nil {
		h.logger.Error(err, "failed to cleanup events")
		WriteError(w, h.logger, err)
		return
	}

	WriteJSONResponse(w, h.logger, http.StatusOK, map[string]string{"message": "Events cleaned up successfully"})
}
