GOPRIVATE ?= "github.com/onmetal/*"
IMG ?= ipxe-service:latest

all: build

build:
	/usr/local/go/bin/go build -o bin/main main.go

run:
	/usr/local/go/bin/go run main.go

test:
	go test -v

image: test
	podman build . -t ${IMG} --build-arg GOPRIVATE=${GOPRIVATE} --build-arg GIT_USER=${GIT_USER} --build-arg GIT_PASSWORD=${GIT_PASSWORD}
