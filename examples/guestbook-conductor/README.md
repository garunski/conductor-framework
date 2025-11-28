# Guestbook Conductor Example

A working example of a conductor built with the [conductor-framework](https://github.com/garunski/conductor-framework) that manages the standard Kubernetes Guestbook application.

## Overview

This example demonstrates how to use the conductor-framework to manage a multi-service Kubernetes application. The Guestbook application is a classic Kubernetes example that consists of:

- **Frontend**: Web server (PHP) serving the guestbook UI
- **Redis Master**: Primary Redis instance for data storage
- **Redis Slave**: Redis replica for read scaling

This conductor manages all three services using the conductor-framework's manifest reconciliation capabilities.

## Structure

```
guestbook-conductor/
├── main.go              # Conductor entry point
├── Dockerfile           # Container build
├── README.md            # This file
├── go.mod               # Go module file
├── manifests/           # Kubernetes manifests
│   ├── frontend/
│   │   ├── deployment.yaml
│   │   └── service.yaml
│   ├── redis-master/
│   │   ├── deployment.yaml
│   │   └── service.yaml
│   └── redis-slave/
│       ├── deployment.yaml
│       └── service.yaml
└── deploy/              # Deployment configurations
    ├── conductor.yaml
    ├── build.sh
    ├── up.sh
    └── down.sh
```

## Quick Start

### Prerequisites

- Go 1.25.4+
- Docker (or compatible container runtime)
- Kubernetes cluster (local or remote)
- kubectl configured to access your cluster

### Building

Build the conductor binary:

```bash
cd examples/guestbook-conductor
go build -o guestbook-conductor .
```

Or build the Docker image:

```bash
cd examples/guestbook-conductor
./deploy/build.sh
```

### Running Locally

Run the conductor locally (requires kubeconfig access):

```bash
./guestbook-conductor
```

The conductor will:
1. Load manifests from the embedded `manifests/` directory
2. Start a web server on port 8081 (configurable via `PORT` env var)
3. Provide a web UI at `http://localhost:8081`
4. Manage Kubernetes resources based on the manifests

### Deploying to Kubernetes

Deploy the conductor to your Kubernetes cluster:

```bash
cd examples/guestbook-conductor/deploy
./up.sh
```

This will:
1. Build the Docker image (if not already built)
2. Deploy the conductor to the `guestbook-conductor` namespace
3. Set up RBAC, PVC, and all necessary resources

To remove the conductor:

```bash
cd examples/guestbook-conductor/deploy
./down.sh
```

## Configuration

The conductor can be configured via environment variables:

- `PORT` - HTTP server port (default: `8081`)
- `BADGER_DATA_PATH` - Path for BadgerDB storage (default: `/data/badger`)
- `RECONCILE_INTERVAL` - Reconciliation interval (default: `30s`)
- `AUTO_DEPLOY` - Enable automatic deployment (default: `false`)
- `LOG_RETENTION_DAYS` - Event log retention in days (default: `7`)
- `VERSION` - Application version (default: `dev`)

## Using the Conductor

### Web UI

Once the conductor is running, access the web UI at `http://localhost:8081` (or port-forward if running in Kubernetes):

```bash
kubectl port-forward -n guestbook-conductor svc/guestbook-conductor 8081:8081
```

The web UI provides:
- Service health monitoring
- Deployment controls (deploy, update, remove)
- Event logs and filtering
- Parameter management

### REST API

The conductor exposes a REST API for programmatic access:

```bash
# Deploy all services
curl -X POST http://localhost:8081/api/up

# Deploy specific services
curl -X POST http://localhost:8081/api/up \
  -H "Content-Type: application/json" \
  -d '{"services": ["frontend", "redis-master", "redis-slave"]}'

# Get service health
curl http://localhost:8081/api/services/health

# Get events
curl http://localhost:8081/api/events?limit=10
```

See the [framework documentation](../../docs/api.md) for complete API reference.

## Manifest Structure

Manifests are organized by service in the `manifests/` directory. Each service has:
- `deployment.yaml` - Kubernetes Deployment resource
- `service.yaml` - Kubernetes Service resource

Manifests use Go template syntax for parameter substitution:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{if .NamePrefix}}{{.NamePrefix}}-{{end}}frontend
  namespace: {{.Namespace}}
spec:
  replicas: {{.Replicas}}
  # ...
```

Parameters can be configured via:
- DeploymentParameters CRD
- Framework defaults
- API overrides

## Customization

### Modifying Manifests

Edit the YAML files in `manifests/` to customize the Guestbook application. The framework will automatically pick up changes when you rebuild.

### Adding Services

To add a new service:
1. Create a new directory in `manifests/` (e.g., `manifests/new-service/`)
2. Add `deployment.yaml` and `service.yaml` files
3. Rebuild the conductor

### Changing Images

Update the image tags in the manifest files or use the DeploymentParameters CRD to override image tags without rebuilding.

## Reference

This example is based on the standard Kubernetes Guestbook application. For more information:

- [Kubernetes Guestbook Example](https://kubernetes.io/docs/tutorials/stateless-application/guestbook/)
- [Conductor Framework Documentation](../../README.md)
- [Framework User Guide](../../docs/guide.md)
- [Framework API Reference](../../docs/api.md)

## Troubleshooting

### Conductor Not Starting

Check the conductor logs:

```bash
kubectl logs -f -n guestbook-conductor deployment/guestbook-conductor
```

### Services Not Deploying

1. Check conductor events: `curl http://localhost:8081/api/events/errors`
2. Verify RBAC permissions
3. Check Kubernetes cluster connectivity

### Image Pull Errors

Ensure the Guestbook images are accessible:
- `gcr.io/google_samples/gb-frontend:v5`
- `gcr.io/google_samples/gb-redisslave:v3`
- `redis:6.0.5`

If using a private registry, update the image references in the manifests.

## License

This example follows the same license as the conductor-framework.

