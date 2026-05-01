# Variables
IMAGE_NAME := cert-manager-webhook-bunny
IMAGE_TAG  := local

.PHONY: all build clean test vet tidy fmt container-build container-run

all: clean tidy fmt vet test build

build:
	CGO_ENABLED=0 go build -o webhook -ldflags '-w -extldflags "-static"' .

test:
	go test -v ./...

vet:
	go vet ./...

tidy:
	go mod tidy
	go mod verify

fmt:
	go fmt ./...

clean:
	$(RM) webhook

container-build:
	buildah build -t $(IMAGE_NAME):$(IMAGE_TAG) -f Containerfile .

container-run:
	podman run --rm -it --read-only --security-opt=no-new-privileges $(IMAGE_NAME):$(IMAGE_TAG)