all: build

build:
	/usr/local/go/bin/go build -o bin/main main.go

run:
	/usr/local/go/bin/go run main.go

