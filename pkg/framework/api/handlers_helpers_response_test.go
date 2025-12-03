package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteYAMLResponse(t *testing.T) {
	handler, err := newTestHandler(t)
	if err != nil {
		t.Fatalf("newTestHandler() error = %v", err)
	}

	yamlData := []byte(`apiVersion: v1
kind: Service
metadata:
  name: test`)

	w := httptest.NewRecorder()

	WriteYAMLResponse(w, handler.logger, yamlData)

	if w.Code != http.StatusOK {
		t.Errorf("WriteYAMLResponse() status code = %v, want %v", w.Code, http.StatusOK)
	}

	// WriteYAMLResponse uses "application/yaml" not "application/x-yaml"
	if w.Header().Get("Content-Type") != "application/yaml" {
		t.Errorf("WriteYAMLResponse() Content-Type = %v, want application/yaml", w.Header().Get("Content-Type"))
	}

	body := w.Body.String()
	if !strings.Contains(body, "kind: Service") {
		t.Error("WriteYAMLResponse() did not write YAML content")
	}
}

