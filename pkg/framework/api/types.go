package api

import "time"

type HealthStatus struct {
	Status     string                     `json:"status"`
	Version    string                     `json:"version,omitempty"`
	Timestamp  time.Time                  `json:"timestamp"`
	Components map[string]ComponentStatus `json:"components,omitempty"`
}

type ComponentStatus struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type ServiceStatus struct {
	Name        string    `json:"name"`
	Namespace   string    `json:"namespace"`
	Port        int       `json:"port"`
	Status      string    `json:"status"`
	Endpoint    string    `json:"endpoint,omitempty"`
	LastChecked time.Time `json:"lastChecked"`
}

type StatusResponse struct {
	Services []ServiceStatus `json:"services"`
}

type EnvVar struct {
	Name   string `json:"name"`
	Value  string `json:"value,omitempty"`
	Source string `json:"source,omitempty"`
}

type ServiceDetailsResponse struct {
	Name string   `json:"name"`
	Env  []EnvVar `json:"env"`
}

type ServiceInfo struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Port      int    `json:"port"`
	Installed bool   `json:"installed"`
}

type ServiceListResponse struct {
	Services []ServiceInfo `json:"services"`
}

type ClusterRequirement struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"` // "pass", "fail", "warning"
	Message     string `json:"message,omitempty"`
	Required    bool   `json:"required"`
}

type ClusterRequirementsResponse struct {
	Requirements []ClusterRequirement `json:"requirements"`
	Overall      string               `json:"overall"` // "pass", "fail", "warning"
}

type DeploymentRequest struct {
	Services []string `json:"services,omitempty"`
}

