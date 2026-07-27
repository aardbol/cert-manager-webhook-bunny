IMAGE_NAME := cert-manager-webhook-bunny
IMAGE_TAG  := local

GO_VERSION ?= 1.26.5
CGO_ENABLED ?= 0
GOOS ?= linux
GOARCH ?= amd64
LDFLAGS ?= -w -extldflags "-static"

HELM_CHART_DIR := deploy/helm

.PHONY: all
all: clean tidy fmt vet test build

.PHONY: build
build:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o webhook -ldflags '$(LDFLAGS)' .

.PHONY: test
test:
	go test -v ./...

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
	helm unittest $(HELM_CHART_DIR)

.PHONY: helm-test
helm-test: helm-lint helm-unittest

.PHONY: container-build
container-build:
	buildah build \
		--build-arg GO_VERSION=$(GO_VERSION) \
		-t $(IMAGE_NAME):$(IMAGE_TAG) \
		-f Containerfile .

.PHONY: container-run
container-run:
	podman run --rm -it --read-only --security-opt=no-new-privileges $(IMAGE_NAME):$(IMAGE_TAG)

.PHONY: release-notes
release-notes:
	@tag="$(TAG)"; \
	if [ -z "$$tag" ]; then echo "ERROR: TAG is required. Usage: make release-notes TAG=<version>" >&2; exit 1; fi; \
	version=$${tag#v}; \
	awk \
		'/^## \['"$$version"'\]/ { flag = 1; next } \
		/^## \[/ { if ( flag ) { exit; } } \
		flag { if ( n ) { print prev; } n++; prev = $$0 }' \
		CHANGELOG.md