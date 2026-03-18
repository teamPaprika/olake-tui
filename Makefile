VERSION ?= v0.2.0-direct
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build run test clean lint install

build:
	go build $(LDFLAGS) -o bin/olake-tui ./cmd/olake-tui/

run:
	go run ./cmd/olake-tui/

test:
	go test ./...

clean:
	rm -rf bin/

lint:
	golangci-lint run

install:
	go install $(LDFLAGS) ./cmd/olake-tui/
