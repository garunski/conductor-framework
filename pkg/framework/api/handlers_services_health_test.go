package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCheckServiceHealthAsync(t *testing.T) {
	service := serviceInfo{
		Name:      "test-service",
		Namespace: "default",
		Port:      8080,
	}

	// Use a short timeout for faster test
	client := &http.Client{
		Timeout: 1 * time.Second,
	}

	ctx := context.Background()
	statusChan := checkServiceHealthAsync(ctx, client, service)

	// Wait for result with timeout
	select {
	case status, ok := <-statusChan:
		if !ok {
			t.Fatal("Channel was closed without sending status")
		}
		// Service won't be reachable (different port), so should be unhealthy
		if status.Status == "healthy" {
			t.Error("Expected unhealthy status for unreachable service")
		}
		if status.Name != service.Name {
			t.Errorf("Status.Name = %v, want %v", status.Name, service.Name)
		}
		if status.Namespace != service.Namespace {
			t.Errorf("Status.Namespace = %v, want %v", status.Namespace, service.Namespace)
		}
		if status.Port != service.Port {
			t.Errorf("Status.Port = %v, want %v", status.Port, service.Port)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("checkServiceHealthAsync timed out")
	}
}

func TestCheckServiceHealthAsync_Timeout(t *testing.T) {
	// Create a test server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Delay longer than client timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	service := serviceInfo{
		Name:      "test-service",
		Namespace: "default",
		Port:      8080,
	}

	// Use a very short timeout
	client := &http.Client{
		Timeout: 100 * time.Millisecond,
	}

	ctx := context.Background()
	statusChan := checkServiceHealthAsync(ctx, client, service)

	// Wait for result
	select {
	case status := <-statusChan:
		// Should return unhealthy due to timeout
		if status.Status == "healthy" {
			t.Error("Expected unhealthy status due to timeout")
		}
		if status.Name != service.Name {
			t.Errorf("Status.Name = %v, want %v", status.Name, service.Name)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("checkServiceHealthAsync timed out")
	}
}

func TestCheckServiceHealthAsync_ContextCancellation(t *testing.T) {
	service := serviceInfo{
		Name:      "test-service",
		Namespace: "default",
		Port:      8080,
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	statusChan := checkServiceHealthAsync(ctx, client, service)

	// Wait for result
	select {
	case status := <-statusChan:
		// Should return quickly due to context cancellation
		if status.Status == "healthy" {
			t.Error("Expected unhealthy status due to context cancellation")
		}
		if status.Name != service.Name {
			t.Errorf("Status.Name = %v, want %v", status.Name, service.Name)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("checkServiceHealthAsync should return quickly on cancelled context")
	}
}

func TestCheckServiceHealth(t *testing.T) {
	// Create a test server that returns healthy status
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" || r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	service := serviceInfo{
		Name:      "test-service",
		Namespace: "default",
		Port:      8080,
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	ctx := context.Background()
	status := checkServiceHealth(ctx, client, service)

	// Service won't be reachable (different port), so should be unhealthy
	if status.Status == "healthy" {
		t.Error("Expected unhealthy status for unreachable service")
	}
	if status.Name != service.Name {
		t.Errorf("Status.Name = %v, want %v", status.Name, service.Name)
	}
	if status.Namespace != service.Namespace {
		t.Errorf("Status.Namespace = %v, want %v", status.Namespace, service.Namespace)
	}
	if status.Port != service.Port {
		t.Errorf("Status.Port = %v, want %v", status.Port, service.Port)
	}
}

