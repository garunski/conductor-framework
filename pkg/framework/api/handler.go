package api

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/go-logr/logr"

	apperrors "github.com/garunski/conductor-framework/pkg/framework/errors"
	"github.com/garunski/conductor-framework/pkg/framework/events"
	"github.com/garunski/conductor-framework/pkg/framework/crd"
	"github.com/garunski/conductor-framework/pkg/framework/reconciler"
	"github.com/garunski/conductor-framework/pkg/framework/store"
)

type Handler struct {
	logger          logr.Logger
	reconcileCh     chan string
	reconciler      *reconciler.Reconciler
	appName         string
	version         string
	templates       *template.Template
	store           *store.ManifestStore
	eventStore      *events.Storage
	parameterClient *crd.Client
	manifestFS      embed.FS
	manifestRoot    string
}

func NewHandler(store *store.ManifestStore, eventStore *events.Storage, logger logr.Logger, reconcileCh chan string, rec *reconciler.Reconciler, appName, version string, parameterClient *crd.Client, customTemplateFS *embed.FS, manifestFS embed.FS, manifestRoot string) (*Handler, error) {
	tmpl, err := loadTemplates(customTemplateFS)
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	// Default appName if not provided
	if appName == "" {
		appName = "Conductor"
	}

	h := &Handler{
		logger:          logger,
		reconcileCh:     reconcileCh,
		reconciler:      rec,
		appName:         appName,
		version:         version,
		templates:       tmpl,
		store:           store,
		eventStore:      eventStore,
		parameterClient: parameterClient,
		manifestFS:      manifestFS,
		manifestRoot:    manifestRoot,
	}

	return h, nil
}

func (h *Handler) renderTemplate(w http.ResponseWriter, name string, data interface{}) error {
	// Ensure AppName and AppVersion are always available in template context
	templateData := make(map[string]interface{})
	if dataMap, ok := data.(map[string]interface{}); ok {
		for k, v := range dataMap {
			templateData[k] = v
		}
	} else if data != nil {
		templateData["Data"] = data
	}
	templateData["AppName"] = h.appName
	templateData["AppVersion"] = h.version
	// Add cache busting timestamp for static assets
	templateData["CacheBust"] = time.Now().Unix()
	
	if err := h.templates.ExecuteTemplate(w, name, templateData); err != nil {
		return fmt.Errorf("failed to execute template %s: %w", name, err)
	}
	return nil
}

func (h *Handler) parseJSONRequest(r *http.Request, v interface{}) error {
	err := json.NewDecoder(r.Body).Decode(v)
	if err != nil {
		if syntaxErr, ok := err.(*json.SyntaxError); ok {
			return fmt.Errorf("%w: invalid request body: JSON syntax error at position %d: %w", apperrors.ErrInvalidRequest, syntaxErr.Offset, syntaxErr)
		}
		if unmarshalTypeErr, ok := err.(*json.UnmarshalTypeError); ok {
			return fmt.Errorf("%w: invalid request body: JSON type error for field %s: expected %s, got %s", apperrors.ErrInvalidRequest, unmarshalTypeErr.Field, unmarshalTypeErr.Type, unmarshalTypeErr.Value)
		}
		return fmt.Errorf("%w: invalid request body: %w", apperrors.ErrInvalidRequest, err)
	}
	return nil
}

