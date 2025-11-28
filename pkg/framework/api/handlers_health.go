package api

import (
	"net/http"
	"time"
)

func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	status := HealthStatus{
		Status:    "healthy",
		Version:   h.version,
		Timestamp: time.Now(),
	}

	WriteJSONResponse(w, h.logger, http.StatusOK, status)
}

func (h *Handler) Readyz(w http.ResponseWriter, r *http.Request) {
	status := HealthStatus{
		Status:     "healthy",
		Version:    h.version,
		Timestamp:  time.Now(),
		Components: make(map[string]ComponentStatus),
	}

	if h.store != nil {
		_ = h.store.List()
		status.Components["database"] = ComponentStatus{Status: "healthy"}
	} else {
		status.Components["database"] = ComponentStatus{
			Status:  "unhealthy",
			Message: "Store not initialized",
		}
		status.Status = "unhealthy"
	}

	if h.reconciler != nil {
		if !h.reconciler.IsReady() {
			status.Components["manager"] = ComponentStatus{
				Status:  "not_ready",
				Message: "Manager not ready",
			}
			status.Status = "unhealthy"
		} else {
			status.Components["manager"] = ComponentStatus{Status: "ready"}
		}
	}

	if h.eventStore != nil {

		_, err := h.eventStore.GetRecentErrors(1)
		if err != nil {
			status.Components["eventStore"] = ComponentStatus{
				Status:  "unavailable",
				Message: err.Error(),
			}
		} else {
			status.Components["eventStore"] = ComponentStatus{Status: "available"}
		}
	} else {
		status.Components["eventStore"] = ComponentStatus{
			Status:  "unavailable",
			Message: "Event store not initialized",
		}
	}

	statusCode := http.StatusOK
	if status.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	WriteJSONResponse(w, h.logger, statusCode, status)
}
