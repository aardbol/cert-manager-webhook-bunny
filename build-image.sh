#!/usr/bin/env bash
set -euo pipefail

VERSION="${VERSION:-unknown}"
COMMIT="${COMMIT:-unknown}"
TREE_STATE="${TREE_STATE:-unknown}"
TARGETARCH="${TARGETARCH:-amd64}"
IMAGE="${IMAGE:-cert-manager-webhook-bunny}"
IMAGE_TAG="${IMAGE_TAG:-latest}"

BASE_IMAGE="gcr.io/distroless/static-debian13:nonroot"

echo "Building ${IMAGE}:${IMAGE_TAG} (arch=${TARGETARCH}, version=${VERSION})"

CGO_ENABLED=0 GOOS=linux GOARCH="${TARGETARCH}" go build -ldflags="-w -extldflags '-static'" -o webhook .

CTR=$(buildah from "${BASE_IMAGE}")

buildah copy "${CTR}" webhook /webhook

buildah config --os linux --arch "${TARGETARCH}" "${CTR}"
buildah config --label "org.opencontainers.image.title=cert-manager-webhook-bunny" "${CTR}"
buildah config --label "org.opencontainers.image.description=cert-manager webhook for bunny.net DNS" "${CTR}"
buildah config --label "org.opencontainers.image.licenses=Apache-2.0" "${CTR}"
buildah config --label "org.opencontainers.image.version=${VERSION}" "${CTR}"

buildah config --entrypoint '["/webhook"]' "${CTR}"

buildah commit --format docker "${CTR}" "${IMAGE}:${IMAGE_TAG}"
buildah rm "${CTR}"

rm -f webhook

echo "Built ${IMAGE}:${IMAGE_TAG}"
