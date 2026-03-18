.PHONY: build run test clean lint install

build:
	go build -o bin/olake-tui ./cmd/olake-tui/

run:
	go run ./cmd/olake-tui/

test:
	go test ./...

clean:
	rm -rf bin/

lint:
	golangci-lint run

install:
	go install ./cmd/olake-tui/
