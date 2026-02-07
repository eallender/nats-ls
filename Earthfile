# SPDX-License-Identifier: Apache-2.0
# Copyright Evan Allender
VERSION 0.8
ARG --global VERSION=0.1.0

# Run CI checks
ci:
    BUILD +fmt
    BUILD +vet
    BUILD +lint
    BUILD +build-all

build-all:
    BUILD +build-linux-amd64
    BUILD +build-linux-arm64
    BUILD +build-darwin-amd64
    BUILD +build-darwin-arm64

deps:
    FROM golang:1.24.5-alpine
    WORKDIR /app
    COPY go.mod go.sum ./
    RUN go mod download

source:
    FROM +deps
    COPY cmd ./cmd
    COPY internal ./internal

build-linux-amd64:
    FROM +source
    RUN GOOS=linux GOARCH=amd64 go build -ldflags="-X github.com/eallender/nats-ls/internal/config.Version=$VERSION" -o nls-linux-amd64 ./cmd/nls
    SAVE ARTIFACT nls-linux-amd64

build-linux-arm64:
    FROM +source
    RUN GOOS=linux GOARCH=arm64 go build -ldflags="-X github.com/eallender/nats-ls/internal/config.Version=$VERSION" -o nls-linux-arm64 ./cmd/nls
    SAVE ARTIFACT nls-linux-arm64

build-darwin-amd64:
    FROM +source
    RUN GOOS=darwin GOARCH=amd64 go build -ldflags="-X github.com/eallender/nats-ls/internal/config.Version=$VERSION" -o nls-darwin-amd64 ./cmd/nls
    SAVE ARTIFACT nls-darwin-amd64

build-darwin-arm64:
    FROM +source
    RUN GOOS=darwin GOARCH=arm64 go build -ldflags="-X github.com/eallender/nats-ls/internal/config.Version=$VERSION" -o nls-darwin-arm64 ./cmd/nls
    SAVE ARTIFACT nls-darwin-arm64

lint:
    FROM +source
    RUN apk add --no-cache git
    RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    RUN golangci-lint run --timeout=5m ./...

fmt:
    FROM +source
    RUN test -z "$(gofmt -l .)"

vet:
    FROM +source
    RUN go vet ./...

tidy:
    FROM +source
    RUN go mod tidy
    RUN git diff --exit-code go.mod go.sum
