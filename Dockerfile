# Build the manager binary
FROM --platform=$BUILDPLATFORM golang:1.25.0 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
# Copy the go module manifests
COPY go.mod go.mod
COPY go.sum go.sum
# Cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go sources
COPY main.go main.go
COPY api/ api/
COPY pkg/ pkg/
COPY internal/ internal/
COPY crds/ crds/
COPY Makefile Makefile

# Run tests and build
RUN make envtest \
 && CGO_ENABLED=0 KUBEBUILDER_ASSETS="/workspace/bin/k8s/current" go test ./... \
 && CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager main.go

# Create final image
FROM alpine:3.22
WORKDIR /
ENV GNUPGHOME=/tmp
ENTRYPOINT ["/usr/local/bin/manager"]

RUN apk --no-cache add ca-certificates gnupg \
 && update-ca-certificates

COPY --from=builder /workspace/manager /usr/local/bin/

USER 65534:65534