package api

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

func checkServiceHealth(ctx context.Context, client *http.Client, svc serviceInfo) ServiceStatus {
	status := ServiceStatus{
		Name:        svc.Name,
		Namespace:   svc.Namespace,
		Port:        svc.Port,
		Status:      "unknown",
		LastChecked: time.Now(),
	}

	healthPaths := []string{"/health", "/healthz", "/readyz"}
	url := fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", svc.Name, svc.Namespace, svc.Port)

	for _, path := range healthPaths {

		select {
		case <-ctx.Done():
			return status
		default:
		}

		fullURL := url + path
		req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
		if err != nil {
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			status.Status = "healthy"
			status.Endpoint = path
			return status
		}
	}

	status.Status = "unhealthy"
	return status
}

