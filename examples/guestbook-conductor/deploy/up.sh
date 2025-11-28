#!/bin/bash
# Bootstrap script for conductor deployment
# This script applies all necessary YAML files in the correct order

set -e

# Configuration
IMAGE_TAG=${IMAGE_TAG:-latest}
NAMESPACE=${NAMESPACE:-guestbook-conductor}
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "Bootstrapping conductor..."
echo "  Image tag: $IMAGE_TAG"
echo "  Namespace: $NAMESPACE"
echo ""

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo "Error: kubectl is not installed or not in PATH"
    exit 1
fi

# Check if cluster is accessible
if ! kubectl cluster-info &> /dev/null; then
    echo "Error: Cannot connect to Kubernetes cluster"
    exit 1
fi

# Check if Docker image exists locally
IMAGE_NAME=${IMAGE_NAME:-guestbook-conductor}
if ! docker image inspect "${IMAGE_NAME}:${IMAGE_TAG}" &> /dev/null; then
    echo "⚠ Warning: Docker image ${IMAGE_NAME}:${IMAGE_TAG} not found locally"
    echo ""
    echo "Build the image first:"
    echo "  cd $(dirname "$SCRIPT_DIR") && IMAGE_TAG=${IMAGE_TAG} ./deploy/build.sh"
    echo ""
    echo "Or if the image exists in a registry, ensure it's accessible."
    echo ""
    read -p "Continue anyway? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

echo "Step 1: Applying conductor deployment with image tag: $IMAGE_TAG"
# Replace IMAGE_TAG_PLACEHOLDER with actual image tag
sed "s|IMAGE_TAG_PLACEHOLDER|${IMAGE_TAG}|g" "${SCRIPT_DIR}/conductor.yaml" | kubectl apply -f -

echo ""
echo "Step 2: Waiting for conductor deployment to be available..."
if kubectl wait --for=condition=available --timeout=300s deployment/guestbook-conductor -n "$NAMESPACE" 2>/dev/null; then
    echo "✓ Conductor deployment is available"
else
    echo "⚠ Warning: Conductor deployment may not be ready yet"
    echo "  Check status with: kubectl get pods -n $NAMESPACE"
fi

echo ""
echo "Step 3: Checking conductor pod status..."
kubectl get pods -n "$NAMESPACE" -l app=guestbook-conductor

echo ""
echo "Bootstrap complete!"
echo ""
echo "To check conductor logs:"
echo "  kubectl logs -f -n $NAMESPACE deployment/guestbook-conductor"
echo ""
echo "To check conductor health:"
echo "  kubectl port-forward -n $NAMESPACE svc/guestbook-conductor 8081:8081"
echo "  curl http://localhost:8081/healthz"
echo "  curl http://localhost:8081/readyz"

