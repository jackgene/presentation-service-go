.DEFAULT_GOAL := build

.PHONY: build
build:
	go build -o dist/presentation-service cmd/server/server.go
