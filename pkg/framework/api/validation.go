package api

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	apperrors "github.com/garunski/conductor-framework/pkg/framework/errors"
	"github.com/garunski/conductor-framework/pkg/framework/events"
	"gopkg.in/yaml.v3"
)

var (
	namespaceRegex    = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	resourceNameRegex = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`)
)

func ValidateNamespace(name string) error {
	if name == "" {
		return fmt.Errorf("%w: namespace cannot be empty", apperrors.ErrInvalidNamespace)
	}
	if len(name) > 63 {
		return fmt.Errorf("%w: namespace must be 63 characters or less", apperrors.ErrInvalidNamespace)
	}
	if !namespaceRegex.MatchString(name) {
		return fmt.Errorf("%w: namespace must be a valid RFC 1123 label (lowercase alphanumeric characters or '-', and must start and end with an alphanumeric character)", apperrors.ErrInvalidNamespace)
	}
	return nil
}

func ValidateResourceName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: resource name cannot be empty", apperrors.ErrInvalidServiceName)
	}
	if len(name) > 253 {
		return fmt.Errorf("%w: resource name must be 253 characters or less", apperrors.ErrInvalidServiceName)
	}
	if !resourceNameRegex.MatchString(name) {
		return fmt.Errorf("%w: resource name must be a valid RFC 1123 subdomain", apperrors.ErrInvalidServiceName)
	}
	return nil
}

func ValidateKey(key string) error {
	if key == "" {
		return fmt.Errorf("%w: key cannot be empty", apperrors.ErrInvalid)
	}

	if len(key) > 512 {
		return fmt.Errorf("%w: key must be 512 characters or less", apperrors.ErrInvalid)
	}
	return nil
}

func ValidateYAML(data []byte) error {
	var obj map[string]interface{}
	if err := yaml.Unmarshal(data, &obj); err != nil {
		return fmt.Errorf("%w: invalid YAML: failed to parse YAML: %w", apperrors.ErrInvalidYAML, err)
	}

	if _, ok := obj["apiVersion"]; !ok {
		return fmt.Errorf("%w: missing apiVersion", apperrors.ErrInvalid)
	}
	if _, ok := obj["kind"]; !ok {
		return fmt.Errorf("%w: missing kind", apperrors.ErrInvalid)
	}
	metadata, ok := obj["metadata"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("%w: missing metadata", apperrors.ErrInvalid)
	}
	if _, ok := metadata["name"].(string); !ok {
		return fmt.Errorf("%w: missing metadata.name", apperrors.ErrInvalid)
	}

	return nil
}

func ParseQueryParams(r *http.Request) (events.EventFilters, error) {
	return ParseEventQueryParams(r.URL.Query())
}

func ParseEventQueryParams(queryParams map[string][]string) (events.EventFilters, error) {
	filters := events.EventFilters{}

	if resource := getFirstQueryParam(queryParams, "resource"); resource != "" {
		if err := ValidateKey(resource); err != nil {
			return filters, fmt.Errorf("%w: invalid resource parameter: %w", apperrors.ErrInvalid, err)
		}
		filters.ResourceKey = resource
	}

	if typeStr := getFirstQueryParam(queryParams, "type"); typeStr != "" {
		eventType := events.EventType(typeStr)
		if eventType != events.EventTypeError && eventType != events.EventTypeSuccess &&
			eventType != events.EventTypeInfo && eventType != events.EventTypeWarning {
			return filters, fmt.Errorf("%w: invalid event type: %s (must be one of: error, success, info, warning)", apperrors.ErrInvalid, typeStr)
		}
		filters.Type = eventType
	}

	if sinceStr := getFirstQueryParam(queryParams, "since"); sinceStr != "" {
		t, err := time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			return filters, fmt.Errorf("%w: invalid since parameter format (use RFC3339): %w", apperrors.ErrInvalid, err)
		}
		filters.Since = t
	}

	if untilStr := getFirstQueryParam(queryParams, "until"); untilStr != "" {
		t, err := time.Parse(time.RFC3339, untilStr)
		if err != nil {
			return filters, fmt.Errorf("%w: invalid until parameter format (use RFC3339): %w", apperrors.ErrInvalid, err)
		}
		filters.Until = t
	}

	if limitStr := getFirstQueryParam(queryParams, "limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			return filters, fmt.Errorf("%w: invalid limit parameter: must be a positive integer", apperrors.ErrInvalid)
		}
		if limit > 1000 {
			return filters, fmt.Errorf("%w: limit cannot exceed 1000", apperrors.ErrInvalid)
		}
		filters.Limit = limit
	} else {
		filters.Limit = 100
	}

	if offsetStr := getFirstQueryParam(queryParams, "offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			return filters, fmt.Errorf("%w: invalid offset parameter: must be a non-negative integer", apperrors.ErrInvalid)
		}
		filters.Offset = offset
	}

	return filters, nil
}

func getFirstQueryParam(queryParams map[string][]string, key string) string {
	if values, ok := queryParams[key]; ok && len(values) > 0 {
		return values[0]
	}
	return ""
}

