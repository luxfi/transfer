.PHONY: build run test test-race lint clean

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

build:
	go build -ldflags="-s -w -X main.version=$(VERSION)" -o bin/transferd ./cmd/transferd/

run: build
	./bin/transferd

test:
	go test -count=1 ./...

test-race:
	go test -race -count=1 ./...

lint:
	go vet ./...

clean:
	rm -rf bin/
