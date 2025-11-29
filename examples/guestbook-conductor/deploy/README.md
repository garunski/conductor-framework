# Conductor Deployment Guide

This directory contains YAML files and scripts for deploying and managing the conductor in a Kubernetes cluster.

## Files

- `conductor.yaml` - Complete conductor deployment (Namespace, RBAC, PVC, Deployment, Service, CRD)
- `build.sh` - Script to build the conductor Docker image
- `up.sh` - Script to deploy the conductor
- `down.sh` - Script to remove the conductor and associated resources

## Quick Start

### Build Conductor Image

First, build the Docker image:

```bash
# Build with auto-generated version (dev-local-{timestamp} or dev-local-{git-hash})
./build.sh

# Build with custom tag (overrides auto-generation)
IMAGE_TAG=v1.0.0 ./build.sh
```

**Versioning Scheme:**
- By default, the build script automatically generates a version tag in the format `dev-local-{timestamp}` or `dev-local-{git-hash}` if in a git repository
- The generated version is stored in `.conductor-version` file in the conductor directory
- This ensures each build has a unique, traceable version and avoids "latest" tag confusion
- You can override the version by setting the `IMAGE_TAG` environment variable

### Deploy Conductor

After building the image, deploy it:

```bash
# Using version from .conductor-version file (created during build)
./up.sh

# Using a custom image tag (overrides version file)
export IMAGE_TAG=dev-local-20250128-143022
./up.sh

# Or inline:
IMAGE_TAG=dev-local-20250128-143022 ./up.sh

# Using custom namespace
NAMESPACE=my-conductor ./up.sh
```

**Version Resolution:**
- The `up.sh` script reads the version from `.conductor-version` file by default (created during build)
- If the file doesn't exist, it generates a new timestamp-based version
- You can override by setting the `IMAGE_TAG` environment variable

### Remove Conductor

```bash
# Remove conductor and all resources
./down.sh

# Using custom namespace
NAMESPACE=my-conductor ./down.sh
```

### Option 2: Manual Application

```bash
# Replace IMAGE_TAG_PLACEHOLDER with your image tag, then apply
sed 's|IMAGE_TAG_PLACEHOLDER|local|g' conductor.yaml | kubectl apply -f -

# Or edit conductor.yaml manually to replace IMAGE_TAG_PLACEHOLDER with your tag, then apply
kubectl apply -f conductor.yaml
```

## Configuration

### Image Tag

The conductor deployment uses versioned image tags to avoid "latest" tag confusion.

**Default Behavior:**
- Build script (`build.sh`) automatically generates a version tag: `dev-local-{timestamp}` or `dev-local-{git-hash}`
- Version is saved to `.conductor-version` file
- Deploy script (`up.sh`) reads from `.conductor-version` file by default

**Using Specific Versions:**

1. **Automatic (Recommended)**: Just build and deploy:
   ```bash
   ./build.sh    # Generates version and saves to .conductor-version
   ./up.sh       # Uses version from .conductor-version
   ```

2. **With custom tag**: Override with environment variable:
   ```bash
   export IMAGE_TAG=dev-local-20250128-143022
   ./up.sh
   ```
   Or inline: `IMAGE_TAG=dev-local-20250128-143022 ./up.sh`

3. **Manual**: Edit `conductor.yaml` and replace `IMAGE_TAG_PLACEHOLDER` with your tag, or use sed:
   ```bash
   sed 's|IMAGE_TAG_PLACEHOLDER|dev-local-20250128-143022|g' conductor.yaml | kubectl apply -f -
   ```

**Viewing Current Version:**
```bash
# Check version file
cat conductor/.conductor-version

# Or check built images
docker images | grep guestbook-conductor
```

### Storage

The conductor uses a PersistentVolumeClaim (`conductor-data`) for BadgerDB data persistence. The PVC:
- Size: 1Gi
- Storage class: Uses cluster default (omitted from spec to auto-detect)
  - In Rancher Desktop, this is typically `local-path`
  - To use a specific storage class, add `storageClassName: <name>` to the PVC spec
- Access mode: ReadWriteOnce

To modify storage settings, edit the PersistentVolumeClaim section in `conductor.yaml`.

### Environment Variables

The conductor container uses the following environment variables (all have defaults):

- `BADGER_DATA_PATH`: Path for BadgerDB data (default: `/data/badger`)
- `PORT`: HTTP server port (default: `8081`)
- `RECONCILE_INTERVAL`: Reconciliation interval (default: `30s`)

These can be modified in the Deployment section of `conductor.yaml`.

## Verification

After deploying, verify the conductor is running:

```bash
# Check pod status
kubectl get pods -n guestbook-conductor

# Check deployment status
kubectl get deployment -n guestbook-conductor

# View conductor logs
kubectl logs -f -n guestbook-conductor deployment/guestbook-conductor

# Test health endpoints
kubectl port-forward -n guestbook-conductor svc/guestbook-conductor 8081:8081
# In another terminal:
curl http://localhost:8081/healthz
curl http://localhost:8081/readyz
```

## Troubleshooting

### Pod Not Starting

1. Check pod status: `kubectl get pods -n guestbook-conductor`
2. Check pod events: `kubectl describe pod -n guestbook-conductor <pod-name>`
3. Check logs: `kubectl logs -n guestbook-conductor <pod-name>`

### Image Pull Errors

- **Build the image first**: Run `./build.sh` to build the Docker image
- **Rancher Desktop Container Runtime**: Rancher Desktop can use either `containerd` or `dockerd`:
  - **If using containerd**: Build images with `nerdctl` instead of `docker`:
    ```bash
    nerdctl build -t guestbook-conductor:latest ..
    ```
    Or import Docker image into containerd:
    ```bash
    docker save guestbook-conductor:latest | nerdctl --namespace k8s.io load
    ```
  - **If using dockerd**: Regular `docker build` should work
  - Check your runtime in Rancher Desktop: Preferences → Container Engine
- **Image Pull Policy**: The deployment uses `imagePullPolicy: IfNotPresent` to use local images
- Verify image exists:
  - Docker: `docker images | grep guestbook-conductor`
  - Containerd: `nerdctl --namespace k8s.io images | grep guestbook-conductor`
- If using a different registry, ensure the image is pushed and accessible

### PVC Not Binding

- Check storage class: `kubectl get storageclass`
- Check PVC status: `kubectl get pvc -n guestbook-conductor`
- Check PVC events: `kubectl describe pvc guestbook-conductor-data -n guestbook-conductor`
- If no default storage class exists, add `storageClassName: <name>` to the PVC spec in `conductor.yaml`

### RBAC Permission Issues

- Verify ServiceAccount exists: `kubectl get sa -n guestbook-conductor`
- Verify ClusterRoleBinding: `kubectl get clusterrolebinding guestbook-conductor`
- Check conductor logs for permission errors

## Uninstalling

To remove the conductor, use the `down.sh` script:

```bash
./down.sh
```

This will:
1. Delete the conductor deployment, service, and PVC
2. Delete RBAC resources
3. Optionally delete the namespace (with confirmation)

**Note**: The `down.sh` script will prompt before deleting the namespace. PVC data will be preserved unless you explicitly delete it or choose to delete the namespace.

### Manual Removal

If you prefer to remove resources manually:

```bash
# Delete all conductor resources (uses version from .conductor-version or environment)
./down.sh

# Or with specific version
IMAGE_TAG=dev-local-20250128-143022 ./down.sh

# Or delete by namespace (removes everything)
kubectl delete namespace guestbook-conductor

# Note: PVC and data will persist unless explicitly deleted
# To delete PVC: kubectl delete pvc guestbook-conductor-data -n guestbook-conductor
```

## Resource Order

The `conductor.yaml` file contains resources in this order:

1. **CustomResourceDefinition** - Defines DeploymentParameters CRD
2. **Namespace** - Created first
3. **ServiceAccount** - Created in the namespace
4. **ClusterRole** - Defines permissions
5. **ClusterRoleBinding** - Binds ServiceAccount to ClusterRole
6. **PersistentVolumeClaim** - Created before deployment
7. **Deployment** - Uses ServiceAccount and PVC
8. **Service** - Exposes the deployment

This order ensures:
- CRD is created before resources that reference it
- Namespace exists before resources that reference it
- RBAC is configured before the deployment that uses it
- PVC is created before the pod that mounts it

