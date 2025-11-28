package api

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	apperrors "github.com/garunski/conductor-framework/pkg/framework/errors"
)

func extractManifestKey(r *http.Request) string {

	rctx := chi.RouteContext(r.Context())
	key := chi.URLParam(r, "key")

	if key == "" && (rctx == nil || len(rctx.URLParams.Keys) == 0 || !hasParam(rctx, "key")) {
		key = chi.URLParam(r, "*")

		if key == "" && (rctx == nil || len(rctx.URLParams.Keys) == 0 || !hasParam(rctx, "*")) {
			key = r.URL.Path
			if len(key) > 0 && key[0] == '/' {
				key = key[1:]
			}

			if len(key) >= 10 && key[:10] == "manifests/" {
				key = key[10:]
			} else if key == "manifests" {
				key = ""
			}
		}
	}
	return key
}

func (h *Handler) ListManifests(w http.ResponseWriter, r *http.Request) {
	manifests := h.store.List()
	WriteJSONResponse(w, h.logger, http.StatusOK, manifests)
}

func (h *Handler) GetManifest(w http.ResponseWriter, r *http.Request) {
	key := extractManifestKey(r)

	if err := ValidateKey(key); err != nil {
		WriteError(w, h.logger, err)
		return
	}

	manifest, ok := h.store.Get(key)
	if !ok {
		WriteError(w, h.logger, fmt.Errorf("%w: manifest %s", apperrors.ErrNotFound, key))
		return
	}

	WriteYAMLResponse(w, h.logger, manifest)
}

func (h *Handler) CreateManifest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}

	if err := h.parseJSONRequest(r, &req); err != nil {
		WriteError(w, h.logger, fmt.Errorf("validation: invalid request body: %w", err))
		return
	}

	if err := ValidateKey(req.Key); err != nil {
		WriteError(w, h.logger, err)
		return
	}

	if err := ValidateYAML([]byte(req.Value)); err != nil {
		WriteError(w, h.logger, err)
		return
	}

	if err := h.store.Create(req.Key, []byte(req.Value)); err != nil {
		h.logger.Error(err, "failed to create manifest", "key", req.Key)
		WriteError(w, h.logger, fmt.Errorf("creation failed: %w", err))
		return
	}

	select {
	case h.reconcileCh <- req.Key:
	default:

	}

	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) UpdateManifest(w http.ResponseWriter, r *http.Request) {
	key := extractManifestKey(r)

	if err := ValidateKey(key); err != nil {
		WriteError(w, h.logger, err)
		return
	}

	var req struct {
		Value string `json:"value"`
	}

	if err := h.parseJSONRequest(r, &req); err != nil {
		WriteError(w, h.logger, err)
		return
	}

	if err := ValidateYAML([]byte(req.Value)); err != nil {
		WriteError(w, h.logger, err)
		return
	}

	if err := h.store.Update(key, []byte(req.Value)); err != nil {
		h.logger.Error(err, "failed to update manifest", "key", key)
		WriteError(w, h.logger, fmt.Errorf("update failed: %w", err))
		return
	}

	select {
	case h.reconcileCh <- key:
	default:
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) DeleteManifest(w http.ResponseWriter, r *http.Request) {
	key := extractManifestKey(r)

	if err := ValidateKey(key); err != nil {
		WriteError(w, h.logger, err)
		return
	}

	if err := h.store.Delete(key); err != nil {
		h.logger.Error(err, "failed to delete manifest", "key", key)
		WriteError(w, h.logger, fmt.Errorf("deletion failed: %w", err))
		return
	}

	select {
	case h.reconcileCh <- key:
	default:
	}

	w.WriteHeader(http.StatusNoContent)
}
