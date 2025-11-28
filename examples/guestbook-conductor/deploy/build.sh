#!/bin/bash
# Build script for conductor Docker image

set -e

# Configuration
IMAGE_TAG=${IMAGE_TAG:-latest}
IMAGE_NAME=${IMAGE_NAME:-guestbook-conductor}
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONDUCTOR_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
FRAMEWORK_DIR="$(cd "${CONDUCTOR_DIR}/../../.." 2>/dev/null && pwd || echo "")"

echo "Building conductor Docker image..."
echo "  Image: ${IMAGE_NAME}:${IMAGE_TAG}"
echo "  Conductor directory: $CONDUCTOR_DIR"
echo ""

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    echo "Error: docker is not installed or not in PATH"
    exit 1
fi

# Check if go.mod exists
if [ ! -f "$CONDUCTOR_DIR/go.mod" ]; then
    echo "Error: go.mod not found in conductor directory: $CONDUCTOR_DIR"
    exit 1
fi

# Check if Dockerfile exists
if [ ! -f "$CONDUCTOR_DIR/Dockerfile" ]; then
    echo "Error: Dockerfile not found in conductor directory: $CONDUCTOR_DIR"
    exit 1
fi

# Check if conductor-framework exists (for replace directive)
if [ -z "$FRAMEWORK_DIR" ] || [ ! -d "$FRAMEWORK_DIR" ]; then
    echo "Warning: conductor-framework not found at expected location"
    echo "  If using replace directive, ensure conductor-framework is available"
fi

# Build the image from conductor directory
cd "$CONDUCTOR_DIR"

# If using replace directive, we need to copy conductor-framework into build context
# Check if replace directive exists in go.mod
if grep -q "replace.*conductor-framework" "$CONDUCTOR_DIR/go.mod"; then
    FRAMEWORK_REL_PATH=$(grep "replace.*conductor-framework" "$CONDUCTOR_DIR/go.mod" | sed -n 's/.*=> \(.*\)/\1/p' | tr -d ' ')
    if [ -n "$FRAMEWORK_REL_PATH" ]; then
        # Resolve relative path from conductor directory
        FRAMEWORK_ABS_PATH=$(cd "$CONDUCTOR_DIR" && cd "$FRAMEWORK_REL_PATH" && pwd)
        if [ -d "$FRAMEWORK_ABS_PATH" ]; then
            echo "Copying conductor-framework into build context..."
            cp -r "$FRAMEWORK_ABS_PATH" "$CONDUCTOR_DIR/conductor-framework"
            TEMP_COPY=true
            # Update replace directive in go.mod for build
            sed -i.bak "s|replace.*conductor-framework.*=>.*|replace github.com/garunski/conductor-framework => ./conductor-framework|g" "$CONDUCTOR_DIR/go.mod"
        fi
    fi
fi

# Build the image
echo "Building Docker image..."
docker build -t "${IMAGE_NAME}:${IMAGE_TAG}" .

# Clean up temporary copy if created
if [ "$TEMP_COPY" = true ]; then
    echo "Cleaning up temporary conductor-framework copy..."
    rm -rf "$CONDUCTOR_DIR/conductor-framework"
    if [ -f "$CONDUCTOR_DIR/go.mod.bak" ]; then
        mv "$CONDUCTOR_DIR/go.mod.bak" "$CONDUCTOR_DIR/go.mod"
    fi
fi

echo ""
echo "Build complete!"
echo ""
echo "Image built: ${IMAGE_NAME}:${IMAGE_TAG}"
echo ""

# Check if using containerd (nerdctl) or dockerd (docker)
if command -v nerdctl &> /dev/null && nerdctl info &> /dev/null; then
    echo "⚠ Rancher Desktop is using containerd runtime"
    echo ""
    echo "To make the image available to Kubernetes, rebuild with nerdctl:"
    echo "  cd $CONDUCTOR_DIR"
    echo "  nerdctl build -t ${IMAGE_NAME}:${IMAGE_TAG} ."
    echo ""
    echo "Or import the Docker image into containerd:"
    echo "  docker save ${IMAGE_NAME}:${IMAGE_TAG} | nerdctl --namespace k8s.io load"
    echo ""
elif command -v docker &> /dev/null && docker info &> /dev/null; then
    echo "✓ Using Docker runtime - image should be available to Kubernetes"
    echo ""
    echo "If Kubernetes still can't see the image, check:"
    echo "  1. Rancher Desktop container engine setting (dockerd vs containerd)"
    echo "  2. Image pull policy is set to IfNotPresent in deploy/conductor.yaml"
    echo ""
fi

echo "To deploy:"
echo "  cd $CONDUCTOR_DIR/deploy && IMAGE_TAG=${IMAGE_TAG} ./up.sh"

