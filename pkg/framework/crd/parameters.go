package crd

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const (
	// DefaultCRDGroup is the default CRD group name
	DefaultCRDGroup = "conductor.io"
	// DefaultCRDVersion is the default CRD version
	DefaultCRDVersion = "v1alpha1"
	// DefaultCRDResource is the default CRD resource name
	DefaultCRDResource = "deploymentparameters"
	// DefaultName is the default name for the DeploymentParameters instance
	DefaultName = "default"
)

// DeploymentParametersSpec represents the spec of DeploymentParameters CRD
// Stored as map[string]interface{} to preserve all fields dynamically
type DeploymentParametersSpec map[string]interface{}

// DeploymentParameters represents the DeploymentParameters CRD
type DeploymentParameters struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              DeploymentParametersSpec `json:"spec,omitempty"`
}

// Client provides methods to interact with DeploymentParameters CRD
type Client struct {
	dynamicClient dynamic.Interface
	logger        logr.Logger
	group         string
	version       string
	resource      string
	gvr           schema.GroupVersionResource
}

// NewClient creates a new DeploymentParameters client
// If group, version, or resource are empty, defaults will be used
func NewClient(dynamicClient dynamic.Interface, logger logr.Logger, group, version, resource string) *Client {
	// Use defaults if not provided
	if group == "" {
		group = DefaultCRDGroup
	}
	if version == "" {
		version = DefaultCRDVersion
	}
	if resource == "" {
		resource = DefaultCRDResource
	}

	return &Client{
		dynamicClient: dynamicClient,
		logger:        logger,
		group:         group,
		version:       version,
		resource:      resource,
		gvr: schema.GroupVersionResource{
			Group:    group,
			Version:  version,
			Resource: resource,
		},
	}
}

// GetCRDSchema retrieves the OpenAPI schema from the CRD definition using dynamic client
func (c *Client) GetCRDSchema(ctx context.Context) (map[string]interface{}, error) {
	crdGVR := schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1",
		Resource: "customresourcedefinitions",
	}
	
	crdName := fmt.Sprintf("%s.%s", c.resource, c.group)
	crdInterface := c.dynamicClient.Resource(crdGVR)
	
	obj, err := crdInterface.Get(ctx, crdName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get CRD definition %s: %w", crdName, err)
	}

	// Extract the schema from the CRD
	spec, ok := obj.Object["spec"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("CRD spec not found")
	}

	versions, ok := spec["versions"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("CRD versions not found")
	}

	// Find the version we need
	for _, v := range versions {
		version, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		if version["name"] == c.version {
			schema, ok := version["schema"].(map[string]interface{})
			if !ok {
				continue
			}
			openAPIV3Schema, ok := schema["openAPIV3Schema"].(map[string]interface{})
			if ok {
				return openAPIV3Schema, nil
			}
		}
	}

	return nil, fmt.Errorf("schema not found for version %s in CRD %s", c.version, crdName)
}

// GetSpecSchema extracts the spec schema structure from the CRD definition
// Returns a map representing the structure of the spec that can be used for UI rendering
func (c *Client) GetSpecSchema(ctx context.Context) (map[string]interface{}, error) {
	schema, err := c.GetCRDSchema(ctx)
	if err != nil {
		return nil, err
	}

	// Navigate to spec.properties
	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("schema properties not found")
	}

	spec, ok := properties["spec"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("spec not found in schema properties")
	}

	specProperties, ok := spec["properties"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("spec properties not found")
	}

	// Build a structure map from the schema
	result := make(map[string]interface{})
	
	// Extract global schema
	if global, ok := specProperties["global"].(map[string]interface{}); ok {
		result["global"] = extractSchemaStructure(global)
	}
	
	// Extract services schema
	if services, ok := specProperties["services"].(map[string]interface{}); ok {
		result["services"] = extractSchemaStructure(services)
	}

	return result, nil
}

// extractSchemaStructure recursively extracts the structure from an OpenAPI schema
func extractSchemaStructure(schema map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	// If it has additionalProperties, it's a flexible object
	if additionalProps, ok := schema["additionalProperties"]; ok {
		if additionalProps == true {
			// Allow any properties - return empty map to indicate flexible structure
			return make(map[string]interface{})
		}
		if apMap, ok := additionalProps.(map[string]interface{}); ok {
			// Recursively extract the structure from additionalProperties
			return extractSchemaStructure(apMap)
		}
	}
	
	// Check if it's an array type
	if schemaType, ok := schema["type"].(string); ok && schemaType == "array" {
		// For arrays, extract the items schema if it's an object
		if items, ok := schema["items"].(map[string]interface{}); ok {
			return extractSchemaStructure(items)
		}
		// For arrays of primitives, return nil to indicate it's a primitive array
		return nil
	}
	
	// Extract defined properties
	if properties, ok := schema["properties"].(map[string]interface{}); ok {
		for key, prop := range properties {
			if propMap, ok := prop.(map[string]interface{}); ok {
				// Check if this is a primitive type (string, number, boolean)
				if propType, ok := propMap["type"].(string); ok {
					if propType == "string" || propType == "number" || propType == "integer" || propType == "boolean" {
						// For primitive types, store the type so template knows it's a leaf node
						result[key] = propType
						continue
					}
				}
				// For complex types, recursively extract
				extracted := extractSchemaStructure(propMap)
				if extracted != nil {
					result[key] = extracted
				} else {
					// Fallback: use empty map
					result[key] = make(map[string]interface{})
				}
			} else {
				result[key] = prop
			}
		}
	} else {
		// If no properties but it's an object type, it might be a primitive object
		// Return empty map to indicate it exists but has no defined structure
		if schemaType, ok := schema["type"].(string); ok && schemaType == "object" {
			return make(map[string]interface{})
		}
	}
	
	return result
}

// Get retrieves a DeploymentParameters instance
func (c *Client) Get(ctx context.Context, name, namespace string) (*DeploymentParameters, error) {
	resourceInterface := c.dynamicClient.Resource(c.gvr).Namespace(namespace)
	
	obj, err := resourceInterface.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get DeploymentParameters %s/%s: %w", namespace, name, err)
	}

	return c.unstructuredToDeploymentParameters(obj)
}

// GetSpec retrieves the spec as map[string]interface{} for template rendering
func (c *Client) GetSpec(ctx context.Context, name, namespace string) (map[string]interface{}, error) {
	resourceInterface := c.dynamicClient.Resource(c.gvr).Namespace(namespace)
	
	obj, err := resourceInterface.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return make(map[string]interface{}), nil
		}
		return nil, fmt.Errorf("failed to get DeploymentParameters %s/%s: %w", namespace, name, err)
	}

	// Extract spec directly from unstructured object
	specRaw, ok := obj.Object["spec"]
	if !ok {
		return make(map[string]interface{}), nil
	}

	specMap, ok := specRaw.(map[string]interface{})
	if !ok {
		return make(map[string]interface{}), nil
	}

	return specMap, nil
}

// Create creates a new DeploymentParameters instance
func (c *Client) Create(ctx context.Context, params *DeploymentParameters) error {
	resourceInterface := c.dynamicClient.Resource(c.gvr).Namespace(params.Namespace)
	
	obj := c.deploymentParametersToUnstructured(params)
	
	_, err := resourceInterface.Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create DeploymentParameters %s/%s: %w", params.Namespace, params.Name, err)
	}

	return nil
}

// CreateWithSpec creates a new DeploymentParameters instance with a spec map
func (c *Client) CreateWithSpec(ctx context.Context, name, namespace string, spec map[string]interface{}) error {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   c.group,
		Version: c.version,
		Kind:    "DeploymentParameters",
	})
	obj.SetName(name)
	obj.SetNamespace(namespace)
	obj.Object["spec"] = spec

	resourceInterface := c.dynamicClient.Resource(c.gvr).Namespace(namespace)
	_, err := resourceInterface.Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create DeploymentParameters %s/%s: %w", namespace, name, err)
	}

	return nil
}

// Update updates an existing DeploymentParameters instance
func (c *Client) Update(ctx context.Context, params *DeploymentParameters) error {
	resourceInterface := c.dynamicClient.Resource(c.gvr).Namespace(params.Namespace)
	
	obj := c.deploymentParametersToUnstructured(params)
	
	_, err := resourceInterface.Update(ctx, obj, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update DeploymentParameters %s/%s: %w", params.Namespace, params.Name, err)
	}

	return nil
}

// UpdateSpec updates the spec of an existing DeploymentParameters instance
func (c *Client) UpdateSpec(ctx context.Context, name, namespace string, spec map[string]interface{}) error {
	resourceInterface := c.dynamicClient.Resource(c.gvr).Namespace(namespace)
	
	// Get existing object to preserve metadata
	obj, err := resourceInterface.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get DeploymentParameters %s/%s: %w", namespace, name, err)
	}

	// Update spec
	obj.Object["spec"] = spec

	_, err = resourceInterface.Update(ctx, obj, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update DeploymentParameters %s/%s: %w", namespace, name, err)
	}

	return nil
}

// List lists all DeploymentParameters instances in a namespace
func (c *Client) List(ctx context.Context, namespace string) ([]DeploymentParameters, error) {
	resourceInterface := c.dynamicClient.Resource(c.gvr).Namespace(namespace)
	
	list, err := resourceInterface.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list DeploymentParameters in namespace %s: %w", namespace, err)
	}

	result := make([]DeploymentParameters, 0, len(list.Items))
	for _, item := range list.Items {
		params, err := c.unstructuredToDeploymentParameters(&item)
		if err != nil {
			c.logger.Error(err, "failed to convert unstructured to DeploymentParameters", "name", item.GetName())
			continue
		}
		result = append(result, *params)
	}

	return result, nil
}


// CreateOrUpdate creates or updates a DeploymentParameters instance
func (c *Client) CreateOrUpdate(ctx context.Context, params *DeploymentParameters) error {
	existing, err := c.Get(ctx, params.Name, params.Namespace)
	if err != nil {
		return err
	}

	if existing == nil {
		return c.Create(ctx, params)
	}

	// Update the resource version for optimistic concurrency
	params.ResourceVersion = existing.ResourceVersion
	return c.Update(ctx, params)
}

// Helper functions

func (c *Client) deploymentParametersToUnstructured(params *DeploymentParameters) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   c.group,
		Version: c.version,
		Kind:    "DeploymentParameters",
	})
	obj.SetName(params.Name)
	obj.SetNamespace(params.Namespace)
	if params.ResourceVersion != "" {
		obj.SetResourceVersion(params.ResourceVersion)
	}

	// Spec is already a map[string]interface{}, use it directly
	if params.Spec != nil {
		obj.Object["spec"] = params.Spec
	} else {
		obj.Object["spec"] = make(map[string]interface{})
	}

	return obj
}


func (c *Client) unstructuredToDeploymentParameters(obj *unstructured.Unstructured) (*DeploymentParameters, error) {
	params := &DeploymentParameters{
		TypeMeta: metav1.TypeMeta{
			APIVersion: fmt.Sprintf("%s/%s", c.group, c.version),
			Kind:       "DeploymentParameters",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            obj.GetName(),
			Namespace:       obj.GetNamespace(),
			ResourceVersion: obj.GetResourceVersion(),
		},
	}

	// Extract spec directly as map[string]interface{} to preserve all fields
	specRaw, ok := obj.Object["spec"]
	if !ok {
		params.Spec = make(map[string]interface{})
		return params, nil
	}

	specMap, ok := specRaw.(map[string]interface{})
	if !ok {
		params.Spec = make(map[string]interface{})
		return params, nil
	}

	// Deep copy the spec map to avoid sharing references
	params.Spec = deepCopyMap(specMap)

	return params, nil
}

// deepCopyMap creates a deep copy of a map[string]interface{}
func deepCopyMap(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{})
	for k, v := range src {
		dst[k] = deepCopyValue(v)
	}
	return dst
}

// deepCopyValue creates a deep copy of a value
func deepCopyValue(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		return deepCopyMap(val)
	case []interface{}:
		dst := make([]interface{}, len(val))
		for i, item := range val {
			dst[i] = deepCopyValue(item)
		}
		return dst
	default:
		// For primitives (string, int, bool, etc.), return as-is
		return v
	}
}
