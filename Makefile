IMAGE_NAME := cert-manager-webhook-bunny
IMAGE_TAG  := local

GO_VERSION ?= 1.25
CGO_ENABLED ?= 0
GOOS ?= linux
GOARCH ?= amd64
LDFLAGS ?= -w -extldflags "-static"

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

.PHONY: container-build
container-build:
	buildah build \
		--build-arg GO_VERSION=$(GO_VERSION) \
		-t $(IMAGE_NAME):$(IMAGE_TAG) \
		-f Containerfile .

.PHONY: container-run
container-run:
	podman run --rm -it --read-only --security-opt=no-new-privileges $(IMAGE_NAME):$(IMAGE_TAG)