#!/bin/bash
set -e

echo "=========================================="
echo "  Docker Multi-Arch Build Script"
echo "=========================================="
echo ""

echo "Select target platform:"
echo "  1) amd64 (Intel/AMD x86_64)"
echo "  2) arm64 (Apple Silicon, Raspberry Pi, ARM servers)"
echo ""
read -p "Enter choice [1-2]: " choice

case $choice in
  1)
    PLATFORM="linux/amd64"
    TAG="medicaments-api:amd64"
    echo "Building for amd64..."
    ;;
  2)
    PLATFORM="linux/arm64"
    TAG="medicaments-api:arm64"
    echo "Building for arm64..."
    ;;
  *)
    echo "Invalid choice"
    exit 1
    ;;
esac

echo ""
echo "Platform: $PLATFORM"
echo "Tag: $TAG"
echo ""

docker buildx build \
  --platform "$PLATFORM" \
  --tag "$TAG" \
  --load \
  .

echo ""
echo "Build complete!"
echo "Image: $TAG"
echo ""
echo "To run:"
echo "  docker run -p 8000:8000 $TAG"
