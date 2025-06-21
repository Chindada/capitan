#!/bin/bash

set -e
VERSION=v0.0

# Gen version
CURRENT_COMMIT=$(git rev-parse HEAD)
echo "{" >internal/version/core.json
echo "  \"version\": \"$VERSION\"," >>internal/version/core.json
echo "  \"commit\": \"$CURRENT_COMMIT\"" >>internal/version/core.json
echo "}" >>internal/version/core.json

# Build binary
mage prod linux build

# Build SRV
docker buildx build --secret id=GH_PAT \
    -t ghcr.io/chindada/capitan:$VERSION \
    -f ./docker/srv.dockerfile .

# Clean up
docker system prune --volumes -f
