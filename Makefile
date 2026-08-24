GO_VERSION ?= 1.26.5
CGO_ENABLED ?= 0
GOOS ?= linux
GOARCH ?= amd64
LDFLAGS ?= -w -extldflags "-static"

TAG ?= $(shell git describe --tags)
COMMIT = $(shell git log --format="%h" -n 1)
TREE_STATE = $(shell git diff --quiet && echo 'clean' || echo 'dirty')
TARGETARCH ?= amd64

CONTAINER_REPO ?= ghcr.io/aardbol/cert-manager-webhook-bunny
IMAGE_TAG ?= local

HELM_CHART_DIR := deploy/helm
HELM_CHART_VERSION ?= $(shell yq .version $(HELM_CHART_DIR)/Chart.yaml)

.PHONY: all
all: clean tidy fmt vet test build

.PHONY: build
build:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o webhook -ldflags '$(LDFLAGS)' .

.PHONY: test
test:
	go test -v ./...

.PHONY: test-integration
test-integration:
	go test -tags integration -v ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: tidy
tidy:
	go mod tidy
	go mod verify

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: clean
clean:
	$(RM) webhook

.PHONY: helm-lint
helm-lint:
	helm lint $(HELM_CHART_DIR)

.PHONY: helm-unittest
helm-unittest:
	@if ! helm plugin list | grep -q unittest > /dev/null 2>&1; then \
		helm plugin install https://github.com/helm-unittest/helm-unittest.git; \
	fi
	helm unittest $(HELM_CHART_DIR)

.PHONY: helm-test
helm-test: helm-lint helm-unittest

.PHONY: helm-package
helm-package:
	mkdir -p dist
	helm package $(HELM_CHART_DIR) --destination dist

.PHONY: helm-release-notes
helm-release-notes:
	@awk \
		'/^## \['$(HELM_CHART_VERSION)'\]/ { flag = 1; next } \
		/^## \[/ { if ( flag ) { exit; } } \
		flag { if ( n ) { print prev; } n++; prev = $$0 } \
		END { if ( flag && n ) { print prev } }' \
		deploy/helm/CHANGELOG.md

.PHONY: container-build
container-build:
	TARGETARCH=$(TARGETARCH) VERSION=$(TAG) COMMIT=$(COMMIT) TREE_STATE=$(TREE_STATE) \
	IMAGE=$(CONTAINER_REPO) IMAGE_TAG=$(IMAGE_TAG) ./build-image.sh

.PHONY: push-image
push-image:
	@echo "==> Pushing image $(CONTAINER_REPO):$(IMAGE_TAG)"
ifdef DIGEST_FILE
	buildah push --digestfile "$(DIGEST_FILE)" "$(CONTAINER_REPO):$(IMAGE_TAG)" "docker://$(CONTAINER_REPO):$(IMAGE_TAG)"
else
	buildah push "$(CONTAINER_REPO):$(IMAGE_TAG)"
endif

.PHONY: container-run
container-run:
	podman run --rm -it --read-only --security-opt=no-new-privileges $(CONTAINER_REPO):$(IMAGE_TAG)

.PHONY: release-notes
release-notes: CHANGELOG_HEADER = ^\#\# \[
release-notes: CHANGELOG_VERSION = $(subst v,,$(TAG))
release-notes:
	@awk \
		'/${CHANGELOG_HEADER}${CHANGELOG_VERSION}/ { flag = 1; next } \
		/${CHANGELOG_HEADER}/ { if ( flag ) { exit; } } \
		flag { if ( n ) { print prev; } n++; prev = $$0 } \
		END { if ( flag && n ) { print prev } }' \
		CHANGELOG.md