BIN     := sessions-control
PKG     := ./cmd/sessions-control
OUT     := bin/$(BIN)
PREFIX  := $(HOME)/.local/bin

.PHONY: all build run test vet fmt tidy install clean

all: build

build:
	@mkdir -p bin
	go build -o $(OUT) $(PKG)

run:
	go run $(PKG)

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w cmd internal

tidy:
	go mod tidy

install: build
	install -d $(PREFIX)
	install -m 0755 $(OUT) $(PREFIX)/$(BIN)

clean:
	rm -rf bin
