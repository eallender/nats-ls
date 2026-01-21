# SPDX-License-Identifier: Apache-2.0
# Copyright Evan Allender
VERSION 0.8
FROM golang:1.24.5-alpine
WORKDIR /workspace

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
    COPY go.mod go.sum ./
    RUN go mod download

build-linux-amd64:
    FROM +deps
    COPY . .
    RUN GOOS=linux GOARCH=amd64 go build -o nls-linux-amd64 .
    SAVE ARTIFACT nls-linux-amd64

build-linux-arm64:
    FROM +deps
    COPY . .
    RUN GOOS=linux GOARCH=arm64 go build -o nls-linux-arm64 .
    SAVE ARTIFACT nls-linux-arm64

build-darwin-amd64:
    FROM +deps
    COPY . .
    RUN GOOS=darwin GOARCH=amd64 go build -o nls-darwin-amd64 .
    SAVE ARTIFACT nls-darwin-amd64

build-darwin-arm64:
    FROM +deps
    COPY . .
    RUN GOOS=darwin GOARCH=arm64 go build -o nls-darwin-arm64 .
    SAVE ARTIFACT nls-darwin-arm64

lint:
    FROM +deps
    COPY . .
    RUN apk add --no-cache git
    RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    RUN golangci-lint run --timeout=5m ./...

fmt:
    FROM +deps
    COPY . .
    RUN test -z "$(gofmt -l .)"

vet:
    FROM +deps
    COPY . .
    RUN go vet ./...

tidy:
    FROM +deps
    COPY . .
    RUN go mod tidy
    RUN git diff --exit-code go.mod go.sum
