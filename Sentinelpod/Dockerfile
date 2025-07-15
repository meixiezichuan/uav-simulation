FROM golang:1.20 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
ENV GOPROXY=https://goproxy.cn,direct
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY sentinel/ sentinel/
COPY common/ common/

# Build
RUN GOOS=linux GOARCH=amd64 go build -ldflags="-extldflags=-static" -a -o sentinelagent main.go
# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM alpine:latest
WORKDIR /
COPY --from=builder /workspace/sentinelagent .
EXPOSE 9898
ENTRYPOINT ["/sentinelagent"]