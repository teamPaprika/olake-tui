# Build stage
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)" \
    -o /olake-tui ./cmd/olake-tui/

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /olake-tui /usr/local/bin/olake-tui

ENTRYPOINT ["olake-tui"]
