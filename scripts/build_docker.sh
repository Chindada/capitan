#!/bin/bash

set -e
VERSION=v1.0

# Pull the latest sequoia image
rm -rf dist
mkdir -p dist
docker run --rm \
    -v $(pwd)/dist:/app \
    ghcr.io/chindada/sequoia:$VERSION \
    cp -r /usr/share/sequoia/dist/. /app
docker run --rm \
    -v $(pwd)/internal/version:/app \
    ghcr.io/chindada/sequoia:$VERSION \
    mv /usr/share/sequoia/version.json /app/fronted.json

# Gen version
CURRENT_COMMIT=$(git rev-parse HEAD)
echo "{" >internal/version/core.json
echo "  \"version\": \"$VERSION\"," >>internal/version/core.json
echo "  \"commit\": \"$CURRENT_COMMIT\"" >>internal/version/core.json
echo "}" >>internal/version/core.json

# Build binary
mage prod linux build

# Build SRV
docker buildx build \
    -t ghcr.io/chindada/capitan:$VERSION \
    -f ./docker/capitan.dockerfile .

# Clean up
docker system prune --volumes -f
