FROM golang:1.16 as builder

WORKDIR /opt/ipxe

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
ARG GOPRIVATE
ARG GIT_USER
ARG GIT_PASSWORD
RUN if [ ! -z "$GIT_USER" ] && [ ! -z "$GIT_PASSWORD" ]; then \
        printf "machine github.com\n \
            login ${GIT_USER}\n \
            password ${GIT_PASSWORD}\n \
            \nmachine api.github.com\n \
            login ${GIT_USER}\n \
            password ${GIT_PASSWORD}\n" \
            >> ${HOME}/.netrc; \
    fi
RUN go mod download

# Copy the go source
COPY main.go main.go

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o ipxe main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot

WORKDIR /
COPY --from=builder /opt/ipxe/ipxe .
USER nonroot:nonroot

ENTRYPOINT ["/ipxe"]
