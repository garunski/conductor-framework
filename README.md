# conductor-framework

A reusable framework for building Kubernetes operators that manage application deployments through manifest reconciliation.

## Overview

The `conductor-framework` provides a complete foundation for building Kubernetes operators with:

- **Manifest Management**: Load, template, and reconcile Kubernetes manifests
- **Web UI**: Built-in web interface for managing deployments
- **REST API**: Comprehensive API for programmatic access
- **Event Tracking**: Persistent event storage and querying
- **CRD Support**: Custom Resource Definitions for parameter management
- **Reconciliation Loop**: Automatic and manual reconciliation of resources

## Quick Start

### Installation

```bash
go get github.com/garunski/conductor-framework/pkg/framework
```

### Basic Usage

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
    cfg.AppName = "my-conductor"
    cfg.AppVersion = "1.0.0"
    cfg.ManifestFS = manifestFiles
    cfg.ManifestRoot = "manifests"
    
    if err := framework.Run(ctx, cfg); err != nil {
        log.Fatalf("Framework error: %v", err)
    }
}
```

## Features

### Manifest Management

- Load manifests from embedded filesystems
- Template rendering with parameter substitution
- Service-based organization
- Override support via API

### Web UI

- Service health monitoring
- Deployment controls
- Event logs and filtering
- Cluster requirements checking
- Parameter management

### REST API

- Manifest CRUD operations
- Deployment control (up/down/update)
- Event querying and filtering
- Service status and health checks
- Parameter management

### Reconciliation

- Manual reconciliation via API endpoints
- Event tracking for all operations
- Error handling and retry logic

## Configuration

The framework is configured via `framework.Config`:

```go
type Config struct {
    // Application metadata
    AppName    string
    AppVersion string
    
    // Manifest configuration
    ManifestFS       embed.FS
    ManifestRoot     string
    CustomTemplateFS *embed.FS
    
    // Storage configuration
    DataPath string
    
    // Server configuration
    Port string
    
    // Logging configuration
    LogRetentionDays  int
    LogCleanupInterval time.Duration
    
    // CRD configuration
    CRDGroup         string
    CRDVersion       string
    CRDResource      string
}
```

### Environment Variables

All configuration can be overridden via environment variables:

- `VERSION` - Application version (default: "dev")
- `BADGER_DATA_PATH` - Data storage path (default: "/data/badger")
- `PORT` - HTTP server port (default: "8081")
- `LOG_RETENTION_DAYS` - Event log retention (default: 7)
- `LOG_CLEANUP_INTERVAL` - Log cleanup interval (default: "1h")

## Documentation

- [User Guide](docs/guide.md) - Comprehensive usage guide
- [API Reference](docs/api.md) - API documentation
- [Migration Guide](docs/migration.md) - Migrating from operator to framework

## Architecture

The framework consists of several key components:

- **Framework Core** (`pkg/framework/framework.go`) - Main entry point
- **Server** (`pkg/framework/server/`) - HTTP server and lifecycle management
- **Reconciler** (`pkg/framework/reconciler/`) - Kubernetes resource reconciliation
- **Manifest** (`pkg/framework/manifest/`) - Manifest loading and templating
- **Store** (`pkg/framework/store/`) - Manifest storage and indexing
- **Events** (`pkg/framework/events/`) - Event tracking and storage
- **API** (`pkg/framework/api/`) - REST API handlers
- **CRD** (`pkg/framework/crd/`) - Custom Resource Definition client

## License

[Add your license here]

## Contributing

[Add contributing guidelines here]
