FROM --platform=$BUILDPLATFORM golang:1.26 AS build
WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o webhook -ldflags '-w -extldflags "-static"' .

FROM gcr.io/distroless/static-debian13:nonroot

LABEL org.opencontainers.image.title="cert-manager-webhook-bunny" \
      org.opencontainers.image.description="cert-manager webhook for bunny.net DNS" \
      org.opencontainers.image.licenses="Apache-2.0"

COPY --from=build /workspace/webhook /webhook

ENTRYPOINT ["/webhook"]