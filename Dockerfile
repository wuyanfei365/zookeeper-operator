# Build the manager binary
# FROM golang:1.17 as builder
FROM golang:1.17-bullseye as builder

RUN go env -w GOPROXY=https://goproxy.cn && \
    mkdir -p /go/bin && \
    mkdir /go/pkg && \
    mkdir -p /go/src/github.com/wuyanfei365/zookeeper-operator && \
    chmod -R 755 /go && \
    mkdir -p /workspace

WORKDIR /go/src/github.com/wuyanfei365/zookeeper-operator
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
# RUN go mod download

# Copy the go source
COPY cmd cmd/
COPY api api/
COPY pkg pkg/
COPY vendor vendor/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOPATH=/go && \
    go build -a -o /workspace/manager cmd/main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
# FROM gcr.io/distroless/static:nonroot
FROM centos:latest AS final
WORKDIR /
COPY --from=builder /workspace/manager .
USER root

ENTRYPOINT ["/manager"]
