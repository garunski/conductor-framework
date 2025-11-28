# Migration Guide

This guide helps you migrate from the old operator codebase to the new k8s-conductor-framework.

## Overview

The framework refactoring involved:

1. **Package Migration**: Moving from `internal/` to `pkg/framework/`
2. **Naming Changes**: "operator" → "conductor"
3. **API Changes**: New framework entry point and configuration
4. **Module Structure**: New Go module structure

## Package Path Changes

### Old → New

| Old Path | New Path |
|----------|----------|
| `github.com/localmeadow/operator/internal/errors` | `github.com/garunski/conductor-framework/pkg/framework/errors` |
| `github.com/localmeadow/operator/internal/database` | `github.com/garunski/conductor-framework/pkg/framework/database` |
| `github.com/localmeadow/operator/internal/events` | `github.com/garunski/conductor-framework/pkg/framework/events` |
| `github.com/localmeadow/operator/internal/store` | `github.com/garunski/conductor-framework/pkg/framework/store` |
| `github.com/localmeadow/operator/internal/reconciler` | `github.com/garunski/conductor-framework/pkg/framework/reconciler` |
| `github.com/localmeadow/operator/internal/api` | `github.com/garunski/conductor-framework/pkg/framework/api` |
| `github.com/localmeadow/operator/internal/manifest` | `github.com/garunski/conductor-framework/pkg/framework/manifest` |
| `github.com/localmeadow/operator/internal/crd` | `github.com/garunski/conductor-framework/pkg/framework/crd` |
| `github.com/localmeadow/operator/internal/server` | `github.com/garunski/conductor-framework/pkg/framework/server` |

## Main Application Changes

### Old Code

```go
package main

import (
    "github.com/localmeadow/operator/internal/config"
    "github.com/localmeadow/operator/internal/server"
)

func main() {
    cfg := config.Load()
    srv, err := server.NewServer(cfg, logger)
    // ...
}
```

### New Code

```go
package main

import (
    "context"
    "embed"
    
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

## Configuration Changes

### Old Configuration

```go
type Config struct {
    DataPath           string
    Port               string
    ReconcileInterval  time.Duration
    AutoDeploy         bool
    LogRetentionDays   int
    LogCleanupInterval time.Duration
    Version            string
}
```

### New Configuration

```go
type Config struct {
    AppName            string
    AppVersion         string
    ManifestFS         embed.FS
    ManifestRoot       string
    DataPath           string
    Port               string
    ReconcileInterval  time.Duration
    AutoDeploy         bool
    LogRetentionDays   int
    LogCleanupInterval time.Duration
    CRDGroup           string
    CRDVersion         string
    CRDResource        string
    CustomTemplateFS   *embed.FS
}
```

## Manifest Loading Changes

### Old Code

Manifests were loaded internally by the server:

```go
embedded, err := manifest.LoadEmbeddedManifests(operator.ManifestFiles, ctx, parameterGetter)
```

### New Code

Manifests are loaded by the framework before server creation:

```go
cfg.ManifestFS = manifestFiles
cfg.ManifestRoot = "manifests"
// Framework loads manifests automatically
```

## CRD Changes

### Old CRD Group

```yaml
apiVersion: localmeadow.io/v1alpha1
kind: DeploymentParameters
```

### New CRD Group

```yaml
apiVersion: conductor.localmeadow.io/v1alpha1
kind: DeploymentParameters
```

### CRD Client Changes

**Old:**
```go
parameterClient := crd.NewClient(dynamicClient, logger)
```

**New:**
```go
parameterClient := crd.NewClient(dynamicClient, logger, group, version, resource)
// Or use defaults:
parameterClient := crd.NewClient(dynamicClient, logger, "", "", "")
```

## Reconciler Changes

### Old Code

```go
rec, err := reconciler.NewReconciler(
    clientset,
    dynamicClient,
    manifestStore,
    logger,
    eventStore,
)
```

### New Code

```go
rec, err := reconciler.NewReconciler(
    clientset,
    dynamicClient,
    manifestStore,
    logger,
    eventStore,
    appName, // New parameter
)
```

## Server Changes

### Old Code

```go
srv, err := server.NewServer(cfg, logger)
```

### New Code

```go
// Manifests must be pre-loaded
manifests := loadManifests(...)
srv, err := server.NewServer(cfg, logger, manifests)
```

## API Handler Changes

### Old Code

```go
handler, err := api.NewHandler(
    manifestStore,
    eventStore,
    logger,
    reconcileCh,
    rec,
    version,
    parameterClient,
)
```

### New Code

```go
handler, err := api.NewHandler(
    manifestStore,
    eventStore,
    logger,
    reconcileCh,
    rec,
    appName,    // New parameter
    version,
    parameterClient,
    customTemplateFS, // New parameter (can be nil)
)
```

## Manifest Loading Changes

### Old Code

```go
manifests, err := manifest.LoadEmbeddedManifests(files, ctx, parameterGetter)
```

### New Code

```go
manifests, err := manifest.LoadEmbeddedManifests(files, rootPath, ctx, parameterGetter)
// rootPath is new parameter (e.g., "manifests")
```

## Naming Changes

### Application Name

- **Old**: Hardcoded as "operator"
- **New**: Configurable via `Config.AppName` (default: "conductor")

### Field Manager

- **Old**: `FieldManager: "operator"`
- **New**: `FieldManager: r.appName` (configurable)

### CRD Defaults

- **Old**: `GroupVersion = "localmeadow.io/v1alpha1"`
- **New**: `DefaultCRDGroup = "conductor.localmeadow.io"`

## Migration Steps

1. **Update Go Module**
   ```bash
   go get github.com/garunski/conductor-framework/pkg/framework
   ```

2. **Update Imports**
   - Replace all `github.com/localmeadow/operator/internal/*` imports
   - Update to `github.com/garunski/conductor-framework/pkg/framework/*`

3. **Update Main Function**
   - Use `framework.Run()` instead of manual server setup
   - Embed manifests using `//go:embed`
   - Use `framework.DefaultConfig()`

4. **Update Configuration**
   - Add `AppName` and `AppVersion`
   - Add `ManifestFS` and `ManifestRoot`
   - Update CRD group if using custom CRDs

5. **Update CRD Resources**
   - Update CRD group from `localmeadow.io` to `conductor.localmeadow.io`
   - Or configure custom group via `Config.CRDGroup`

6. **Test Migration**
   - Verify all functionality works
   - Check event logs
   - Test API endpoints
   - Verify web UI

## Breaking Changes

1. **Package Paths**: All imports must be updated
2. **Main Function**: Must use `framework.Run()`
3. **Manifest Loading**: Must be done before server creation
4. **CRD Group**: Default changed to `conductor.localmeadow.io`
5. **Reconciler**: Requires `appName` parameter
6. **API Handler**: Requires `appName` and optional `customTemplateFS`

## Example Migration

See `examples/localmeadow-conductor/` for a complete migrated example.

## Getting Help

If you encounter issues during migration:

1. Check the example application
2. Review the API documentation
3. Check event logs for errors
4. Verify all imports are updated

