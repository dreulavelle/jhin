.PHONY: all fmt lint test bench build

all: fmt lint test

fmt:
	go fmt ./...

lint:
	go vet ./...

test:
	go test ./...

bench:
	go test -bench=. -benchmem -run='^$$' ./parser/

build:
	go build -o bin/jhin cmd/jhin/main.go
