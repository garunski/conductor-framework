#!/bin/bash
# Full clear script for conductor deployment
# This script removes the conductor, ALL data (including PVC), and all associated resources

set -e

# Configuration
NAMESPACE=${NAMESPACE:-guestbook-conductor}
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONDUCTOR_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "=========================================="
echo "CONDUCTOR FULL CLEAR"
echo "=========================================="
echo ""
echo "⚠️  WARNING: This will delete EVERYTHING including:"
echo "   - All conductor deployments and pods"
echo "   - All BadgerDB data (PVC will be deleted)"
echo "   - All RBAC resources"
echo "   - All services"
echo "   - The entire namespace: $NAMESPACE"
echo ""
echo "This action CANNOT be undone!"
echo ""

# Confirmation prompt
read -p "Are you sure you want to proceed? Type 'yes' to confirm: " -r
if [[ ! $REPLY == "yes" ]]; then
    echo "Aborted."
    exit 1
fi

echo ""
echo "Starting full cleanup..."
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

# Step 1: Delete deployment and service (if namespace exists)
echo "Step 1: Deleting deployment and service..."
if kubectl get namespace "$NAMESPACE" &> /dev/null; then
    kubectl delete deployment guestbook-conductor -n "$NAMESPACE" --ignore-not-found=true || true
    kubectl delete service guestbook-conductor -n "$NAMESPACE" --ignore-not-found=true || true
    kubectl delete poddisruptionbudget guestbook-conductor -n "$NAMESPACE" --ignore-not-found=true || true
    echo "✓ Deployment and service deleted"
else
    echo "✓ Namespace does not exist, skipping deployment/service deletion"
fi

# Step 2: Delete PVC and PV (this deletes all BadgerDB data)
echo ""
echo "Step 2: Deleting PersistentVolumeClaim and PersistentVolume (all BadgerDB data will be lost)..."
if kubectl get namespace "$NAMESPACE" &> /dev/null; then
    if kubectl get pvc guestbook-conductor-data -n "$NAMESPACE" &> /dev/null; then
        # Get the PV name before deleting PVC
        PV_NAME=$(kubectl get pvc guestbook-conductor-data -n "$NAMESPACE" -o jsonpath='{.spec.volumeName}' 2>/dev/null || echo "")
        
        # Delete PVC (this should trigger PV deletion if reclaim policy is Delete)
        kubectl delete pvc guestbook-conductor-data -n "$NAMESPACE" --ignore-not-found=true
        
        # Wait a moment for PV to be released
        sleep 2
        
        # Force delete PV if it still exists (remove finalizers)
        if [ -n "$PV_NAME" ] && kubectl get pv "$PV_NAME" &> /dev/null; then
            echo "  Force deleting PersistentVolume: $PV_NAME"
            kubectl patch pv "$PV_NAME" -p '{"metadata":{"finalizers":[]}}' --type=merge 2>/dev/null || true
            kubectl delete pv "$PV_NAME" --ignore-not-found=true || true
            echo "✓ PV deleted"
        fi
        
        echo "✓ PVC deleted (all BadgerDB data has been removed)"
    else
        echo "✓ PVC does not exist"
    fi
else
    echo "✓ Namespace does not exist, skipping PVC deletion"
fi

# Step 3: Delete RBAC resources (ClusterRoleBinding and ClusterRole)
echo ""
echo "Step 3: Deleting RBAC resources..."
kubectl delete clusterrolebinding guestbook-conductor --ignore-not-found=true || true
kubectl delete clusterrole guestbook-conductor --ignore-not-found=true || true
echo "✓ RBAC resources deleted"

# Step 4: Delete CRD (CustomResourceDefinition)
echo ""
echo "Step 4: Deleting CustomResourceDefinition..."
kubectl delete crd deploymentparameters.conductor.io --ignore-not-found=true || true
echo "✓ CRD deleted"

# Step 5: Delete namespace (this deletes everything remaining)
echo ""
echo "Step 5: Deleting namespace (this will remove all remaining resources)..."
if kubectl get namespace "$NAMESPACE" &> /dev/null; then
    kubectl delete namespace "$NAMESPACE" --ignore-not-found=true
    echo "✓ Namespace deletion initiated"
    
    # Wait for namespace to be fully deleted
    echo "  Waiting for namespace to be fully deleted..."
    while kubectl get namespace "$NAMESPACE" &> /dev/null; do
        echo "  Still deleting... (this may take a moment)"
        sleep 2
    done
    echo "✓ Namespace fully deleted"
else
    echo "✓ Namespace does not exist"
fi

# Step 6: Clean up local version file
echo ""
echo "Step 6: Cleaning up local files..."
VERSION_FILE="${CONDUCTOR_DIR}/.conductor-version"
if [ -f "$VERSION_FILE" ]; then
    rm -f "$VERSION_FILE"
    echo "✓ Version file deleted"
else
    echo "✓ Version file does not exist"
fi

echo ""
echo "=========================================="
echo "CLEANUP COMPLETE"
echo "=========================================="
echo ""
echo "All conductor resources have been deleted:"
echo "  ✓ Deployment and pods"
echo "  ✓ Services"
echo "  ✓ PersistentVolumeClaim (all data deleted)"
echo "  ✓ RBAC resources (ClusterRole, ClusterRoleBinding)"
echo "  ✓ CustomResourceDefinition"
echo "  ✓ Namespace: $NAMESPACE"
echo "  ✓ Local version file"
echo ""
echo "The conductor has been completely removed from the cluster."
echo "All BadgerDB data has been permanently deleted."
echo ""
echo "To redeploy, run:"
echo "  cd $CONDUCTOR_DIR/deploy && ./up.sh"
echo ""

