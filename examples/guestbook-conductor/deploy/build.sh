#!/bin/bash
# Build script for conductor Docker image

set -e

# Configuration
IMAGE_NAME=${IMAGE_NAME:-guestbook-conductor}
UPDATE_DEPS=${UPDATE_DEPS:-false}
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONDUCTOR_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
ROOT_DIR="$(cd "${CONDUCTOR_DIR}/../.." && pwd)"
VERSION_FILE="${CONDUCTOR_DIR}/.conductor-version"

# Generate version if not provided
if [ -z "$IMAGE_TAG" ]; then
    # Use timestamp with random component for guaranteed uniqueness
    RANDOM_SUFFIX=$(openssl rand -hex 8 2>/dev/null || echo $(($RANDOM * $RANDOM)))
    IMAGE_TAG="dev-$(date +%Y%m%d-%H%M%S)-${RANDOM_SUFFIX}"
fi

echo "Building conductor Docker image..."
echo "  Image: ${IMAGE_NAME}:${IMAGE_TAG}"
echo "  Root directory: $ROOT_DIR"
echo "  Conductor directory: $CONDUCTOR_DIR"
echo ""

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    echo "Error: docker is not installed or not in PATH"
    exit 1
fi

# Check if go.mod exists in root directory (framework root)
if [ ! -f "$ROOT_DIR/go.mod" ]; then
    echo "Error: go.mod not found in root directory: $ROOT_DIR"
    exit 1
fi

# Check if go.mod exists in conductor directory
if [ ! -f "$CONDUCTOR_DIR/go.mod" ]; then
    echo "Error: go.mod not found in conductor directory: $CONDUCTOR_DIR"
    exit 1
fi

# Check if Dockerfile exists
if [ ! -f "$CONDUCTOR_DIR/Dockerfile" ]; then
    echo "Error: Dockerfile not found in conductor directory: $CONDUCTOR_DIR"
    exit 1
fi

# Update dependencies if requested
if [ "$UPDATE_DEPS" = "true" ]; then
    echo "Updating dependencies to latest..."
    cd "$CONDUCTOR_DIR"
    go get -u github.com/garunski/conductor-framework@latest
    go mod tidy
    echo ""
fi

# Run tests before building
echo "Running tests..."
cd "$CONDUCTOR_DIR"
if ! go test ./...; then
    echo "Error: Tests failed. Build aborted."
    exit 1
fi
echo "All tests passed!"
echo ""

# Build the image from root directory (so replace directive works)
echo "Building Docker image..."
cd "$ROOT_DIR"
docker build -f examples/guestbook-conductor/Dockerfile -t "${IMAGE_NAME}:${IMAGE_TAG}" .

# Store version in file for reference
echo "$IMAGE_TAG" > "$VERSION_FILE"

echo ""
echo "Build complete!"
echo ""
echo "Image built: ${IMAGE_NAME}:${IMAGE_TAG}"
echo "Version saved to: $VERSION_FILE"
echo ""

# Check if using containerd (nerdctl) or dockerd (docker)
if command -v nerdctl &> /dev/null && nerdctl info &> /dev/null; then
    echo "⚠ Rancher Desktop is using containerd runtime"
    echo ""
    echo "To make the image available to Kubernetes, rebuild with nerdctl:"
    echo "  cd $CONDUCTOR_DIR"
    echo "  nerdctl build -f Dockerfile -t ${IMAGE_NAME}:${IMAGE_TAG} ."
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
