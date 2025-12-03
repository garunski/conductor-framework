package api

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (h *Handler) SetupRoutes() *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(CORSMiddleware)

	r.Group(func(r chi.Router) {
		r.Use(middleware.Timeout(30 * time.Second))
		r.Get("/", h.HomePage)
		r.Get("/parameters", h.ParametersPage)
		r.Get("/deployments", h.DeploymentsPage)
		r.Get("/logs", h.LogsPage)
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.Timeout(10 * time.Second))
		r.Get("/healthz", h.Healthz)
		r.Get("/readyz", h.Readyz)
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.Timeout(60 * time.Second))
		r.Post("/api/up", h.Up)
		r.Post("/api/down", h.Down)
		r.Post("/api/update", h.Update)
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.Timeout(5 * time.Second))
		r.Get("/api/services", h.ListServices)
		r.Get("/api/services/health", h.Status)
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.Timeout(10 * time.Second))
		r.Get("/api/cluster/requirements", h.ClusterRequirements)
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.Timeout(30 * time.Second))
		r.Get("/api/service/{namespace}/{name}", h.ServiceDetails)
	})

	r.Route("/manifests", func(r chi.Router) {
		r.Use(middleware.Timeout(30 * time.Second))
		r.Get("/", h.ListManifests)
		r.Post("/", h.CreateManifest)
		r.Get("/*", h.GetManifest)
		r.Put("/*", h.UpdateManifest)
		r.Delete("/*", h.DeleteManifest)
	})

	r.Route("/api/events", func(r chi.Router) {
		r.Use(middleware.Timeout(30 * time.Second))
		r.Get("/", h.ListEvents)
		r.Get("/errors", h.GetRecentErrors)
		r.Delete("/", h.CleanupEvents)
		r.Get("/*", h.GetEventsByResource)
	})

	r.Route("/api/parameters", func(r chi.Router) {
		r.Use(middleware.Timeout(30 * time.Second))
		r.Get("/", h.GetParameters)
		r.Post("/", h.UpdateParameters)
		r.Get("/schema", h.GetParametersSchema)
		r.Get("/values", h.GetServiceValues)
		r.Get("/{service}", h.GetServiceParameters)
		r.Get("/instances", h.ListParameterInstances)
		r.Post("/instances", h.CreateParameterInstance)
	})

	// Serve static files (JS, CSS, etc.)
	r.Get("/static/*", h.ServeStatic)

	return r
}

