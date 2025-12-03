package api

import (
	"net/http/httptest"
	"testing"

	"github.com/garunski/conductor-framework/pkg/framework/events"
)

func TestValidateNamespace(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid namespace", "default", false},
		{"valid namespace with dash", "my-namespace", false},
		{"valid namespace with numbers", "namespace123", false},
		{"empty namespace", "", true},
		{"namespace too long", string(make([]byte, 64)), true},
		{"namespace with uppercase", "Default", true},
		{"namespace starting with dash", "-namespace", true},
		{"namespace ending with dash", "namespace-", true},
		{"namespace with underscore", "my_namespace", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNamespace(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateNamespace(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if err != nil && !tt.wantErr {
				t.Errorf("ValidateNamespace(%q) unexpected error: %v", tt.input, err)
			}
		})
	}
}

func TestValidateResourceName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid resource name", "my-resource", false},
		{"valid resource name with dot", "my.resource", false},
		{"valid resource name with subdomain", "my.resource.name", false},
		{"empty resource name", "", true},
		{"resource name too long", string(make([]byte, 254)), true},
		{"resource name with uppercase", "MyResource", true},
		{"resource name starting with dash", "-resource", true},
		{"resource name ending with dash", "resource-", true},
		{"resource name with underscore", "my_resource", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateResourceName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateResourceName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateKey(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid key", "default/Service/test", false},
		{"empty key", "", true},
		{"key too long", string(make([]byte, 513)), true},
		{"valid long key", string(make([]byte, 512)), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateKey(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateKey(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateYAML(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name: "valid YAML",
			input: `apiVersion: v1
kind: Service
metadata:
  name: test-service`,
			wantErr: false,
		},
		{
			name:    "invalid YAML syntax",
			input:   "invalid: yaml: [",
			wantErr: true,
		},
		{
			name: "missing apiVersion",
			input: `kind: Service
metadata:
  name: test-service`,
			wantErr: true,
		},
		{
			name: "missing kind",
			input: `apiVersion: v1
metadata:
  name: test-service`,
			wantErr: true,
		},
		{
			name: "missing metadata",
			input: `apiVersion: v1
kind: Service`,
			wantErr: true,
		},
		{
			name: "missing metadata.name",
			input: `apiVersion: v1
kind: Service
metadata: {}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateYAML([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateYAML() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseQueryParams(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr bool
		check   func(*testing.T, events.EventFilters)
	}{
		{
			name:    "empty query",
			query:   "",
			wantErr: false,
			check: func(t *testing.T, f events.EventFilters) {
				if f.Limit != 100 {
					t.Errorf("ParseQueryParams() default limit = %v, want 100", f.Limit)
				}
			},
		},
		{
			name:    "with resource",
			query:   "resource=default/Service/test",
			wantErr: false,
			check: func(t *testing.T, f events.EventFilters) {
				if f.ResourceKey != "default/Service/test" {
					t.Errorf("ParseQueryParams() resource = %v, want default/Service/test", f.ResourceKey)
				}
			},
		},
		{
			name:    "with type",
			query:   "type=error",
			wantErr: false,
			check: func(t *testing.T, f events.EventFilters) {
				if f.Type != events.EventTypeError {
					t.Errorf("ParseQueryParams() type = %v, want error", f.Type)
				}
			},
		},
		{
			name:    "invalid type",
			query:   "type=invalid",
			wantErr: true,
		},
		{
			name:    "with limit",
			query:   "limit=50",
			wantErr: false,
			check: func(t *testing.T, f events.EventFilters) {
				if f.Limit != 50 {
					t.Errorf("ParseQueryParams() limit = %v, want 50", f.Limit)
				}
			},
		},
		{
			name:    "limit too high",
			query:   "limit=2000",
			wantErr: true,
		},
		{
			name:    "invalid limit",
			query:   "limit=invalid",
			wantErr: true,
		},
		{
			name:    "with offset",
			query:   "offset=10",
			wantErr: false,
			check: func(t *testing.T, f events.EventFilters) {
				if f.Offset != 10 {
					t.Errorf("ParseQueryParams() offset = %v, want 10", f.Offset)
				}
			},
		},
		{
			name:    "negative offset",
			query:   "offset=-1",
			wantErr: true,
		},
		{
			name:    "with since",
			query:   "since=2023-01-01T00:00:00Z",
			wantErr: false,
			check: func(t *testing.T, f events.EventFilters) {
				if f.Since.IsZero() {
					t.Error("ParseQueryParams() since should not be zero")
				}
			},
		},
		{
			name:    "invalid since format",
			query:   "since=invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/events?"+tt.query, nil)
			filters, err := ParseQueryParams(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseQueryParams() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, filters)
			}
		})
	}
}

func TestParseEventQueryParams(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string][]string
		wantErr bool
	}{
		{
			name:    "empty params",
			params:  map[string][]string{},
			wantErr: false,
		},
		{
			name:    "valid params",
			params:  map[string][]string{"resource": {"test"}, "type": {"error"}, "limit": {"50"}},
			wantErr: false,
		},
		{
			name:    "invalid resource key",
			params:  map[string][]string{"resource": {string(make([]byte, 513))}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseEventQueryParams(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseEventQueryParams() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetFirstQueryParam(t *testing.T) {
	tests := []struct {
		name   string
		params map[string][]string
		key    string
		want   string
	}{
		{
			name:   "param exists",
			params: map[string][]string{"test": {"value1", "value2"}},
			key:    "test",
			want:   "value1",
		},
		{
			name:   "param does not exist",
			params: map[string][]string{"other": {"value"}},
			key:    "test",
			want:   "",
		},
		{
			name:   "empty params",
			params: map[string][]string{},
			key:    "test",
			want:   "",
		},
		{
			name:   "empty value array",
			params: map[string][]string{"test": {}},
			key:    "test",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getFirstQueryParam(tt.params, tt.key)
			if got != tt.want {
				t.Errorf("getFirstQueryParam() = %v, want %v", got, tt.want)
			}
		})
	}
}

