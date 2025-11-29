#!/bin/bash
# Down script for conductor deployment
# This script removes the conductor and all associated resources

set -e

# Configuration
NAMESPACE=${NAMESPACE:-guestbook-conductor}
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONDUCTOR_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
VERSION_FILE="${CONDUCTOR_DIR}/.conductor-version"

# Get image tag from environment, version file, or use default
if [ -z "$IMAGE_TAG" ]; then
    if [ -f "$VERSION_FILE" ]; then
        IMAGE_TAG=$(cat "$VERSION_FILE")
    else
        # If no version file and no env var, we can't determine version
        # Try to delete with a wildcard or prompt user
        echo "Warning: No version file found and IMAGE_TAG not set."
        echo "Attempting to delete with any tag (this may not work)..."
        IMAGE_TAG="*"
    fi
fi

echo "Removing conductor..."
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

echo "Step 1: Deleting conductor and all resources..."
# Delete all conductor resources (namespace, RBAC, deployment, service, PVC)
# Replace placeholder with any value for deletion (doesn't matter)
if [ "$IMAGE_TAG" = "*" ]; then
    # Delete by label selector instead
    kubectl delete -f "${SCRIPT_DIR}/conductor.yaml" --ignore-not-found=true || true
    # Also try to delete by namespace
    kubectl delete namespace "$NAMESPACE" --ignore-not-found=true || true
else
    sed "s|IMAGE_TAG_PLACEHOLDER|${IMAGE_TAG}|g" "${SCRIPT_DIR}/conductor.yaml" | kubectl delete -f - --ignore-not-found=true
fi

echo ""
echo "Step 2: Checking for remaining resources in namespace..."
# Check if namespace still exists and has resources
if kubectl get namespace "$NAMESPACE" &> /dev/null; then
    echo "  Namespace $NAMESPACE still exists"
    echo "  Remaining resources:"
    kubectl get all -n "$NAMESPACE" 2>/dev/null || echo "    (none)"
    echo ""
    read -p "Delete namespace $NAMESPACE? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        kubectl delete namespace "$NAMESPACE" --ignore-not-found=true
        echo "✓ Namespace deleted"
    else
        echo "  Namespace preserved (may contain other resources)"
    fi
else
    echo "✓ Namespace already deleted or does not exist"
fi

echo ""
echo "Conductor removal complete!"
echo ""
echo "Note: PersistentVolumeClaim data may still exist."
echo "To delete PVC and all data:"
echo "  kubectl delete pvc guestbook-conductor-data -n $NAMESPACE"

