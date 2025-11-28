# k8s-conductor-framework User Guide

This guide provides comprehensive documentation for using the k8s-conductor-framework to build Kubernetes operators.

## Table of Contents

1. [Getting Started](#getting-started)
2. [Configuration](#configuration)
3. [Manifest Management](#manifest-management)
   - [Template Functions](#available-template-functions)
4. [Web UI](#web-ui)
5. [REST API](#rest-api)
6. [Reconciliation](#reconciliation)
7. [Custom Resource Definitions](#custom-resource-definitions)
8. [Deployment](#deployment)
9. [Troubleshooting](#troubleshooting)

## Getting Started

### Installation

Add the framework to your Go module:

```bash
go get github.com/garunski/conductor-framework/pkg/framework
```

### Minimal Example

```go
package main

import (
    "context"
    "embed"
    "log"
    
    "github.com/garunski/conductor-framework/pkg/framework"
)

//go:embed manifests
var manifestFiles embed.FS

func main() {
    ctx := context.Background()
    
    cfg := framework.DefaultConfig()
    cfg.AppName = "my-operator"
    cfg.ManifestFS = manifestFiles
    
    if err := framework.Run(ctx, cfg); err != nil {
        log.Fatalf("Framework error: %v", err)
    }
}
```

## Configuration

### Using DefaultConfig

The easiest way to get started is using `framework.DefaultConfig()`:

```go
cfg := framework.DefaultConfig()
cfg.AppName = "my-operator"
cfg.ManifestFS = myManifests
```

### Custom Configuration

You can customize all aspects of the framework:

```go
import "text/template"

cfg := framework.Config{
    AppName:            "my-operator",
    AppVersion:         "1.0.0",
    ManifestFS:         myManifests,
    ManifestRoot:       "manifests",
    DataPath:           "/data/badger",
    Port:               "8081",
    LogRetentionDays:   7,
    LogCleanupInterval: 1 * time.Hour,
    CRDGroup:           "conductor.localmeadow.io",
    CRDVersion:         "v1alpha1",
    CRDResource:        "deploymentparameters",
    TemplateFuncs:      template.FuncMap{
        // Optional: Add custom template functions
        "myFunc": func(s string) string {
            return "custom-" + s
        },
    },
}
```

### Environment Variables

All configuration can be overridden via environment variables:

```bash
export VERSION=1.0.0
export PORT=8081
```

## Manifest Management

### Manifest Structure

Organize your manifests by service:

```
manifests/
├── redis/
│   ├── deployment.yaml
│   ├── service.yaml
│   └── pvc.yaml
├── postgresql/
│   ├── deployment.yaml
│   └── service.yaml
└── ...
```

### Embedding Manifests

Use Go's `embed` directive to include manifests:

```go
//go:embed manifests
var manifestFiles embed.FS
```

### Template Rendering

Manifests support Go template syntax for parameter substitution with access to 60+ template functions from the Sprig library (used by Helm), plus custom functions.

#### Basic Template Syntax

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.ServiceName}}
spec:
  replicas: {{.Replicas | default 1}}
  template:
    spec:
      containers:
      - name: {{.ServiceName}}
        image: {{.ImageTag | default "latest"}}
```

Parameters are provided via CRD or extracted from manifest defaults.

#### Available Template Functions

The framework provides access to:

1. **Sprig Functions** - 60+ functions from the [Sprig library](https://masterminds.github.io/sprig/) (same library used by Helm)
2. **Built-in Functions** - Framework-specific helper functions
3. **Custom Functions** - User-defined functions via `Config.TemplateFuncs`

##### Sprig Functions

Commonly used Sprig functions for Kubernetes manifests:

**String Manipulation:**
```yaml
name: {{ .ServiceName | upper }}
name: {{ .ServiceName | lower }}
name: {{ .ServiceName | title }}
name: {{ trim "  test  " }}
name: {{ trimPrefix "redis-" "redis-master" }}
name: {{ trimSuffix "-service" "my-service" }}
name: {{ replace "old" "new" "old value" }}
name: {{ contains "test" "testing" }}
```

**Encoding (for Secrets):**
```yaml
apiVersion: v1
kind: Secret
data:
  api-key: {{ "my-secret-value" | b64enc }}
  # Decode: {{ "bXktc2VjcmV0LXZhbHVl" | b64dec }}
```

**Math Operations:**
```yaml
replicas: {{ add .Replicas 1 }}
replicas: {{ sub 5 2 }}
replicas: {{ mul 3 4 }}
replicas: {{ div 10 2 }}
replicas: {{ max 5 10 }}
replicas: {{ min 5 10 }}
```

**List Operations:**
```yaml
env:
{{- range list "ENV1=value1" "ENV2=value2" }}
  - name: {{ . | split "=" | first }}
    value: {{ . | split "=" | last }}
{{- end }}
```

**Dictionary Operations:**
```yaml
labels:
  {{- $labels := dict "app" .ServiceName "version" .ImageTag }}
  {{- $labels = merge $labels .CustomLabels }}
  {{- range $k, $v := $labels }}
  {{ $k }}: {{ $v }}
  {{- end }}
```

**Default Values:**
```yaml
namespace: {{ default "default" .Namespace }}
replicas: {{ default 1 .Replicas }}
image: {{ coalesce .ImageTag .DefaultImage "latest" }}
```

##### Built-in Custom Functions

**uuidv5** - Generate deterministic UUIDs (useful for secrets):

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: {{ .ServiceName }}-secret
data:
  api-key: {{ uuidv5 "6ba7b810-9dad-11d1-80b4-00c04fd430c8" (printf "%s-%s-key" .Namespace .ServiceName) | b64enc }}
```

The `uuidv5` function generates a deterministic UUID v5 from a namespace UUID and name. If the namespace UUID is empty, it defaults to the DNS namespace UUID.

**Built-in Helper Functions:**
```yaml
# Use default value if empty
name: {{ defaultIfEmpty .NamePrefix "default" }}

# Prefix a name
name: {{ prefixName .NamePrefix "service" }}

# Check if prefix exists
{{- if hasPrefix .NamePrefix }}
name: {{ prefixName .NamePrefix .ServiceName }}
{{- end }}
```

##### Custom Template Functions

You can provide custom template functions via `Config.TemplateFuncs`:

```go
import (
    "text/template"
    "github.com/garunski/conductor-framework/pkg/framework"
)

cfg := framework.DefaultConfig()
cfg.TemplateFuncs = template.FuncMap{
    "myCustomFunc": func(s string) string {
        return "custom-" + s
    },
    // Override Sprig functions if needed
    "upper": func(s string) string {
        return "OVERRIDDEN-" + s
    },
}
```

Custom functions have the highest priority and can override Sprig or built-in functions.

**Example Usage:**
```yaml
name: {{ myCustomFunc .ServiceName }}
```

##### Security Note

For security reasons, the `env` and `expandenv` Sprig functions are excluded from the available functions. This prevents templates from accessing environment variables, which could expose sensitive information.

##### Function Reference

For a complete list of available Sprig functions, see the [Sprig documentation](https://masterminds.github.io/sprig/). All functions except `env` and `expandenv` are available.

### Manifest Overrides

Manifests can be overridden via the REST API:

```bash
curl -X POST http://localhost:8081/manifests \
  -H "Content-Type: application/json" \
  -d '{
    "key": "default/Deployment/redis",
    "value": "..."
  }'
```

## Web UI

The framework provides a built-in web UI accessible at `http://localhost:8081`.

### Features

- **Service Health**: Monitor service status and health
- **Deployment Controls**: Deploy, update, or remove services
- **Event Logs**: View reconciliation events and errors
- **Cluster Requirements**: Check cluster compatibility
- **Parameter Management**: Configure deployment parameters

### Custom Templates

You can provide custom HTML templates:

```go
//go:embed custom-templates
var customTemplates embed.FS

cfg := framework.DefaultConfig()
cfg.CustomTemplateFS = &customTemplates
```

## REST API

### Endpoints

#### Health

- `GET /healthz` - Health check
- `GET /readyz` - Readiness check

#### Manifests

- `GET /manifests` - List all manifests
- `GET /manifests/*` - Get specific manifest
- `POST /manifests` - Create manifest override
- `PUT /manifests/*` - Update manifest
- `DELETE /manifests/*` - Delete manifest override

#### Deployment

- `POST /api/up` - Deploy all or selected services
- `POST /api/down` - Remove all or selected services
- `POST /api/update` - Update all or selected services

#### Events

- `GET /api/events` - List events with filtering
- `GET /api/events/*` - Get events for specific resource
- `GET /api/events/errors` - Get recent errors
- `DELETE /api/events` - Cleanup old events

#### Services

- `GET /api/services` - List all services
- `GET /api/services/health` - Get service health status
- `GET /api/service/{namespace}/{name}` - Get service details

#### Parameters

- `GET /api/parameters` - Get deployment parameters
- `POST /api/parameters` - Update deployment parameters
- `GET /api/parameters/{service}` - Get service-specific parameters
- `GET /api/parameters/values` - Get all parameter values

### Example API Usage

```bash
# Deploy all services
curl -X POST http://localhost:8081/api/up

# Deploy specific services
curl -X POST http://localhost:8081/api/up \
  -H "Content-Type: application/json" \
  -d '{"services": ["redis", "postgresql"]}'

# Get events
curl http://localhost:8081/api/events?limit=10

# Get service health
curl http://localhost:8081/api/services/health
```

## Reconciliation

Reconciliation is performed manually via API endpoints. The framework does not support automatic periodic reconciliation.

```bash
# Reconcile specific resource
curl -X POST http://localhost:8081/api/up \
  -H "Content-Type: application/json" \
  -d '{"services": ["redis"]}'
```

### Reconciliation Events

All reconciliation operations generate events:

```bash
# Get reconciliation events
curl http://localhost:8081/api/events?type=info

# Get errors
curl http://localhost:8081/api/events/errors
```

## Custom Resource Definitions

The framework supports CRDs for parameter management.

### CRD Configuration

```go
cfg.CRDGroup = "conductor.localmeadow.io"
cfg.CRDVersion = "v1alpha1"
cfg.CRDResource = "deploymentparameters"
```

Note: Parameter support is always enabled. If Kubernetes is unavailable, the framework will use default parameters for template rendering.

### DeploymentParameters CRD

The framework uses a `DeploymentParameters` CRD to manage deployment parameters:

```yaml
apiVersion: conductor.localmeadow.io/v1alpha1
kind: DeploymentParameters
metadata:
  name: default
  namespace: default
spec:
  global:
    namespace: default
    replicas: 1
  services:
    redis:
      replicas: 3
      imageTag: "7.0"
```

### Using Parameters

Parameters are automatically merged and used during manifest templating.

## Deployment

### Building

```bash
go build -o my-operator .
```

### Docker

```dockerfile
FROM golang:1.25.4-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o conductor .

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /app/conductor /conductor
CMD ["/conductor"]
```

### Kubernetes

See `examples/localmeadow-conductor/deploy/` for complete Kubernetes deployment manifests.

## Troubleshooting

### Common Issues

#### Manifest Loading Errors

- Verify manifest files are properly embedded
- Check `ManifestRoot` matches your directory structure
- Ensure YAML files are valid

#### Reconciliation Failures

- Check Kubernetes cluster connectivity
- Verify RBAC permissions
- Review event logs: `GET /api/events/errors`

#### Web UI Not Loading

- Verify server is running: `GET /healthz`
- Check port configuration
- Review server logs

### Debugging

Enable debug logging:

```go
import "go.uber.org/zap"

zapLog, _ := zap.NewDevelopment()
logger := zapr.NewLogger(zapLog)
```

### Getting Help

- Check event logs via API or Web UI
- Review reconciliation events
- Check Kubernetes resource status

