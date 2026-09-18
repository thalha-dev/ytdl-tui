BINARY := bin/ytdl-tui
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build install test e2e run vet fmt clean

build: vet
	go build -ldflags '$(LDFLAGS)' -o $(BINARY) .

install: build
	install -m 0755 $(BINARY) /usr/local/bin/

test: vet
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

e2e: build
	python3 scripts/e2e_drive.py

run: build
	./$(BINARY)

clean:
	rm -rf bin
.PHONY: screenshots
screenshots: build
	python3 scripts/screenshots/render.py
