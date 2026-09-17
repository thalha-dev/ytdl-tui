BINARY := bin/ytdl-tui
MODULE := github.com/thalha-dev/ytdl-tui
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build install run test vet fmt e2e clean

build:
	go build -ldflags '$(LDFLAGS)' -o $(BINARY) ./cmd/ytdl-tui

install: build
	install -m 0755 $(BINARY) /usr/local/bin/ytdl-tui

run: build
	$(BINARY)

vet:
	go vet ./...

fmt:
	gofmt -w .

test:
	go test -short ./...

e2e: build
	python3 scripts/e2e_drive.py

clean:
	rm -rf bin
