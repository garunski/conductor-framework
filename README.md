# Conductor Framework

A reusable framework for building Kubernetes operators that manage application deployments through manifest reconciliation.

[![Documentation](https://img.shields.io/badge/docs-latest-blue)](https://garunski.github.io/conductor-framework/)
[![Go Reference](https://pkg.go.dev/badge/github.com/garunski/conductor-framework.svg)](https://pkg.go.dev/github.com/garunski/conductor-framework)

## Overview

The `conductor-framework` provides a complete foundation for building Kubernetes operators with minimal boilerplate. It handles manifest management, templating, reconciliation, and provides both a REST API and Web UI out of the box.

The framework is designed for developers who want to build operators that manage multi-service Kubernetes applications without writing all the infrastructure code from scratch.

## Features

- **Manifest Management**: Load, template, and reconcile Kubernetes manifests with 60+ Sprig template functions (same as Helm)
- **REST API & Web UI**: Built-in web interface and comprehensive REST API for managing deployments, monitoring health, and viewing events
- **CRD-Based Parameters**: Manage deployment parameters using Kubernetes Custom Resource Definitions for a native Kubernetes experience
- **Event Tracking**: Persistent event storage with BadgerDB, querying, filtering, and retention policies for all reconciliation operations
- **Reconciliation Loop**: Manual reconciliation via API endpoints with event tracking, error handling, and retry logic
- **Template System**: Go template syntax with Sprig functions, custom functions support, and parameter injection from CRDs or defaults

## Quick Start

### Installation

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

## Documentation

📚 **[Full Documentation](https://garunski.github.io/conductor-framework/)** - Complete documentation site with guides, examples, and API reference.

The documentation includes:
- [Examples](https://garunski.github.io/conductor-framework/examples.html) - Learn by example with the Guestbook Conductor
- [AI Agents](https://garunski.github.io/conductor-framework/agents.html) - How AI agents can interact with the framework
- [Implementation](https://garunski.github.io/conductor-framework/implementation.html) - Deep dive into architecture and components
- [Design Concepts](https://garunski.github.io/conductor-framework/design.html) - Design philosophy and patterns
- [Contributing](https://garunski.github.io/conductor-framework/contributing.html) - How to contribute to the project

## Examples

Check out the [Guestbook Conductor example](examples/guestbook-conductor/) to see a complete working implementation that manages a multi-service Kubernetes application.

```bash
cd examples/guestbook-conductor
go build -o guestbook-conductor .
./guestbook-conductor
```

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
    TemplateFuncs    template.FuncMap // Optional custom template functions
    
    // Storage configuration
    DataPath string
    
    // Server configuration
    Port string
    
    // Logging configuration
    LogRetentionDays    int
    LogCleanupInterval  time.Duration
    
    // CRD configuration
    CRDGroup         string
    CRDVersion       string
    CRDResource      string
}
```

### Environment Variables

Configuration can be overridden via environment variables:

- `VERSION` - Application version (default: "dev")
- `BADGER_DATA_PATH` - Data storage path (default: "/data/badger")
- `PORT` - HTTP server port (default: "8081")
- `LOG_RETENTION_DAYS` - Event log retention (default: 7)
- `LOG_CLEANUP_INTERVAL` - Log cleanup interval (default: "1h")

## Architecture

The framework consists of several key components:

- **Framework Core** (`pkg/framework/framework.go`) - Main entry point
- **Server** (`pkg/framework/server/`) - HTTP server and lifecycle management
- **Reconciler** (`pkg/framework/reconciler/`) - Kubernetes resource reconciliation
- **Manifest** (`pkg/framework/manifest/`) - Manifest loading and templating
- **Store** (`pkg/framework/store/`) - Manifest storage and indexing
- **Events** (`pkg/framework/events/`) - Event tracking and storage
- **API** (`pkg/framework/api/`) - REST API handlers and Web UI
- **CRD** (`pkg/framework/crd/`) - Custom Resource Definition client
- **Database** (`pkg/framework/database/`) - BadgerDB integration

## Requirements

- Go 1.25.4+ (or latest stable version)
- Kubernetes cluster access (local or remote)
- kubectl configured to access your cluster

## License

See [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please see the [Contributing Guide](https://garunski.github.io/conductor-framework/contributing.html) for details on how to contribute.

## Links

- [Documentation](https://garunski.github.io/conductor-framework/)
- [Examples](examples/guestbook-conductor/)
- [API Reference](https://garunski.github.io/conductor-framework/api.html)
