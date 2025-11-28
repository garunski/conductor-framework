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
	DefaultCRDGroup = "conductor.localmeadow.io"
	// DefaultCRDVersion is the default CRD version
	DefaultCRDVersion = "v1alpha1"
	// DefaultCRDResource is the default CRD resource name
	DefaultCRDResource = "deploymentparameters"
	// DefaultName is the default name for the DeploymentParameters instance
	DefaultName = "default"
)

// ResourceRequirements represents resource requests and limits
type ResourceRequirements struct {
	Requests *ResourceList `json:"requests,omitempty"`
	Limits   *ResourceList `json:"limits,omitempty"`
}

// ResourceList represents memory and CPU resources
type ResourceList struct {
	Memory string `json:"memory,omitempty"`
	CPU    string `json:"cpu,omitempty"`
}

// ParameterSet represents a set of deployment parameters
type ParameterSet struct {
	Namespace    string                `json:"namespace,omitempty"`
	NamePrefix   string                `json:"namePrefix,omitempty"`
	Replicas     *int32                `json:"replicas,omitempty"`
	ImageTag     string                `json:"imageTag,omitempty"`
	Resources    *ResourceRequirements `json:"resources,omitempty"`
	StorageSize  string                `json:"storageSize,omitempty"`
	Labels       map[string]string     `json:"labels,omitempty"`
	Annotations  map[string]string     `json:"annotations,omitempty"`
	NodeSelector map[string]string     `json:"nodeSelector,omitempty"`
	Tolerations  []interface{}         `json:"tolerations,omitempty"`
}

// DeploymentParametersSpec represents the spec of DeploymentParameters CRD
type DeploymentParametersSpec struct {
	Global   *ParameterSet            `json:"global,omitempty"`
	Services map[string]*ParameterSet `json:"services,omitempty"`
}

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

// GetMergedParameters returns merged parameters for a specific service
// It merges global defaults with service-specific overrides
func (c *Client) GetMergedParameters(ctx context.Context, serviceName, namespace string) (*ParameterSet, error) {
	params, err := c.Get(ctx, DefaultName, namespace)
	if err != nil {
		return nil, err
	}

	if params == nil {
		// Return default parameter set if CRD doesn't exist
		return &ParameterSet{
			Namespace:  "default",
			NamePrefix: "",
			Replicas:   int32Ptr(1),
		}, nil
	}

	merged := &ParameterSet{}

	// Start with global defaults
	if params.Spec.Global != nil {
		merged = deepCopyParameterSet(params.Spec.Global)
	} else {
		// Set defaults if global is not set
		merged.Namespace = "default"
		merged.NamePrefix = ""
		merged.Replicas = int32Ptr(1)
	}

	// Apply service-specific overrides
	if params.Spec.Services != nil {
		if serviceOverrides, ok := params.Spec.Services[serviceName]; ok && serviceOverrides != nil {
			mergeParameterSet(merged, serviceOverrides)
		}
	}

	return merged, nil
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

func int32Ptr(i int32) *int32 {
	return &i
}

func deepCopyParameterSet(src *ParameterSet) *ParameterSet {
	if src == nil {
		return nil
	}

	dst := &ParameterSet{
		Namespace:   src.Namespace,
		NamePrefix:  src.NamePrefix,
		ImageTag:    src.ImageTag,
		StorageSize: src.StorageSize,
	}

	if src.Replicas != nil {
		dst.Replicas = int32Ptr(*src.Replicas)
	}

	if src.Resources != nil {
		dst.Resources = &ResourceRequirements{}
		if src.Resources.Requests != nil {
			dst.Resources.Requests = &ResourceList{
				Memory: src.Resources.Requests.Memory,
				CPU:    src.Resources.Requests.CPU,
			}
		}
		if src.Resources.Limits != nil {
			dst.Resources.Limits = &ResourceList{
				Memory: src.Resources.Limits.Memory,
				CPU:    src.Resources.Limits.CPU,
			}
		}
	}

	if src.Labels != nil {
		dst.Labels = make(map[string]string)
		for k, v := range src.Labels {
			dst.Labels[k] = v
		}
	}

	if src.Annotations != nil {
		dst.Annotations = make(map[string]string)
		for k, v := range src.Annotations {
			dst.Annotations[k] = v
		}
	}

	if src.NodeSelector != nil {
		dst.NodeSelector = make(map[string]string)
		for k, v := range src.NodeSelector {
			dst.NodeSelector[k] = v
		}
	}

	if src.Tolerations != nil {
		dst.Tolerations = make([]interface{}, len(src.Tolerations))
		copy(dst.Tolerations, src.Tolerations)
	}

	return dst
}

func mergeParameterSet(dst *ParameterSet, src *ParameterSet) {
	if src.Namespace != "" {
		dst.Namespace = src.Namespace
	}
	if src.NamePrefix != "" {
		dst.NamePrefix = src.NamePrefix
	}
	if src.Replicas != nil {
		dst.Replicas = int32Ptr(*src.Replicas)
	}
	if src.ImageTag != "" {
		dst.ImageTag = src.ImageTag
	}
	if src.StorageSize != "" {
		dst.StorageSize = src.StorageSize
	}

	if src.Resources != nil {
		if dst.Resources == nil {
			dst.Resources = &ResourceRequirements{}
		}
		if src.Resources.Requests != nil {
			if dst.Resources.Requests == nil {
				dst.Resources.Requests = &ResourceList{}
			}
			if src.Resources.Requests.Memory != "" {
				dst.Resources.Requests.Memory = src.Resources.Requests.Memory
			}
			if src.Resources.Requests.CPU != "" {
				dst.Resources.Requests.CPU = src.Resources.Requests.CPU
			}
		}
		if src.Resources.Limits != nil {
			if dst.Resources.Limits == nil {
				dst.Resources.Limits = &ResourceList{}
			}
			if src.Resources.Limits.Memory != "" {
				dst.Resources.Limits.Memory = src.Resources.Limits.Memory
			}
			if src.Resources.Limits.CPU != "" {
				dst.Resources.Limits.CPU = src.Resources.Limits.CPU
			}
		}
	}

	if src.Labels != nil {
		if dst.Labels == nil {
			dst.Labels = make(map[string]string)
		}
		for k, v := range src.Labels {
			dst.Labels[k] = v
		}
	}

	if src.Annotations != nil {
		if dst.Annotations == nil {
			dst.Annotations = make(map[string]string)
		}
		for k, v := range src.Annotations {
			dst.Annotations[k] = v
		}
	}

	if src.NodeSelector != nil {
		if dst.NodeSelector == nil {
			dst.NodeSelector = make(map[string]string)
		}
		for k, v := range src.NodeSelector {
			dst.NodeSelector[k] = v
		}
	}

	if src.Tolerations != nil {
		dst.Tolerations = src.Tolerations
	}
}

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

	// Convert spec to unstructured
	spec := make(map[string]interface{})
	if params.Spec.Global != nil {
		spec["global"] = c.parameterSetToMap(params.Spec.Global)
	}
	if params.Spec.Services != nil && len(params.Spec.Services) > 0 {
		services := make(map[string]interface{})
		for k, v := range params.Spec.Services {
			services[k] = c.parameterSetToMap(v)
		}
		spec["services"] = services
	}
	obj.Object["spec"] = spec

	return obj
}

func (c *Client) parameterSetToMap(ps *ParameterSet) map[string]interface{} {
	if ps == nil {
		return nil
	}

	m := make(map[string]interface{})
	if ps.Namespace != "" {
		m["namespace"] = ps.Namespace
	}
	if ps.NamePrefix != "" {
		m["namePrefix"] = ps.NamePrefix
	}
	if ps.Replicas != nil {
		m["replicas"] = *ps.Replicas
	}
	if ps.ImageTag != "" {
		m["imageTag"] = ps.ImageTag
	}
	if ps.StorageSize != "" {
		m["storageSize"] = ps.StorageSize
	}
	if ps.Resources != nil {
		resources := make(map[string]interface{})
		if ps.Resources.Requests != nil {
			requests := make(map[string]interface{})
			if ps.Resources.Requests.Memory != "" {
				requests["memory"] = ps.Resources.Requests.Memory
			}
			if ps.Resources.Requests.CPU != "" {
				requests["cpu"] = ps.Resources.Requests.CPU
			}
			if len(requests) > 0 {
				resources["requests"] = requests
			}
		}
		if ps.Resources.Limits != nil {
			limits := make(map[string]interface{})
			if ps.Resources.Limits.Memory != "" {
				limits["memory"] = ps.Resources.Limits.Memory
			}
			if ps.Resources.Limits.CPU != "" {
				limits["cpu"] = ps.Resources.Limits.CPU
			}
			if len(limits) > 0 {
				resources["limits"] = limits
			}
		}
		if len(resources) > 0 {
			m["resources"] = resources
		}
	}
	if ps.Labels != nil && len(ps.Labels) > 0 {
		m["labels"] = ps.Labels
	}
	if ps.Annotations != nil && len(ps.Annotations) > 0 {
		m["annotations"] = ps.Annotations
	}
	if ps.NodeSelector != nil && len(ps.NodeSelector) > 0 {
		m["nodeSelector"] = ps.NodeSelector
	}
	if ps.Tolerations != nil && len(ps.Tolerations) > 0 {
		m["tolerations"] = ps.Tolerations
	}

	return m
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

	specRaw, ok := obj.Object["spec"]
	if !ok {
		return params, nil
	}

	specMap, ok := specRaw.(map[string]interface{})
	if !ok {
		return params, nil
	}

	if globalRaw, ok := specMap["global"]; ok {
		if globalMap, ok := globalRaw.(map[string]interface{}); ok {
			params.Spec.Global = c.mapToParameterSet(globalMap)
		}
	}

	if servicesRaw, ok := specMap["services"]; ok {
		if servicesMap, ok := servicesRaw.(map[string]interface{}); ok {
			params.Spec.Services = make(map[string]*ParameterSet)
			for k, v := range servicesMap {
				if serviceMap, ok := v.(map[string]interface{}); ok {
					params.Spec.Services[k] = c.mapToParameterSet(serviceMap)
				}
			}
		}
	}

	return params, nil
}

func (c *Client) mapToParameterSet(m map[string]interface{}) *ParameterSet {
	ps := &ParameterSet{}

	if v, ok := m["namespace"].(string); ok {
		ps.Namespace = v
	}
	if v, ok := m["namePrefix"].(string); ok {
		ps.NamePrefix = v
	}
	if v, ok := m["replicas"].(int64); ok {
		ps.Replicas = int32Ptr(int32(v))
	}
	if v, ok := m["imageTag"].(string); ok {
		ps.ImageTag = v
	}
	if v, ok := m["storageSize"].(string); ok {
		ps.StorageSize = v
	}

	if resourcesRaw, ok := m["resources"].(map[string]interface{}); ok {
		ps.Resources = &ResourceRequirements{}
		if requestsRaw, ok := resourcesRaw["requests"].(map[string]interface{}); ok {
			ps.Resources.Requests = &ResourceList{}
			if v, ok := requestsRaw["memory"].(string); ok {
				ps.Resources.Requests.Memory = v
			}
			if v, ok := requestsRaw["cpu"].(string); ok {
				ps.Resources.Requests.CPU = v
			}
		}
		if limitsRaw, ok := resourcesRaw["limits"].(map[string]interface{}); ok {
			ps.Resources.Limits = &ResourceList{}
			if v, ok := limitsRaw["memory"].(string); ok {
				ps.Resources.Limits.Memory = v
			}
			if v, ok := limitsRaw["cpu"].(string); ok {
				ps.Resources.Limits.CPU = v
			}
		}
	}

	if v, ok := m["labels"].(map[string]interface{}); ok {
		ps.Labels = make(map[string]string)
		for k, val := range v {
			if strVal, ok := val.(string); ok {
				ps.Labels[k] = strVal
			}
		}
	}

	if v, ok := m["annotations"].(map[string]interface{}); ok {
		ps.Annotations = make(map[string]string)
		for k, val := range v {
			if strVal, ok := val.(string); ok {
				ps.Annotations[k] = strVal
			}
		}
	}

	if v, ok := m["nodeSelector"].(map[string]interface{}); ok {
		ps.NodeSelector = make(map[string]string)
		for k, val := range v {
			if strVal, ok := val.(string); ok {
				ps.NodeSelector[k] = strVal
			}
		}
	}

	if v, ok := m["tolerations"].([]interface{}); ok {
		ps.Tolerations = v
	}

	return ps
}
