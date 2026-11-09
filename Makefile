BIN       := sessions-control
PKG       := ./cmd/sessions-control
OUT       := bin/$(BIN)
PREFIX    := $(HOME)/.local/bin

VERSION   ?= $(shell git describe --tags --exact-match 2>/dev/null | sed 's/^v//')
ifeq ($(VERSION),)
VERSION   := 0.1.0
endif
DIST      := dist
LDFLAGS   := -s -w -X main.version=$(VERSION)
NFPM      ?= nfpm
PKG_ARCH  ?= amd64
PKG_ARCH_DEB  = $(if $(filter armv7,$(PKG_ARCH)),armhf,$(if $(filter 386,$(PKG_ARCH)),i386,$(PKG_ARCH)))
PKG_ARCH_RPM  = $(if $(filter armv7,$(PKG_ARCH)),armv7hl,$(if $(filter 386,$(PKG_ARCH)),i686,$(PKG_ARCH)))
PKG_ARCH_APK  = $(if $(filter 386,$(PKG_ARCH)),x86,$(PKG_ARCH))
PKG_ARCH_ARCH = $(if $(filter armv7,$(PKG_ARCH)),armv7h,$(if $(filter 386,$(PKG_ARCH)),i686,$(PKG_ARCH)))
PLATFORMS := linux/amd64 linux/arm64 linux/arm/v7 linux/386

.PHONY: all build run test vet fmt tidy install clean \
        dist checksums packages release deb rpm apk archlinux packaging-help

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
	rm -rf bin $(DIST)

# --- Empaquetado -----------------------------------------------------------
# dist/  -> tarballs estáticos para amd64, arm64, armv7 e 386
# *      -> paquetes nativos deb, rpm, apk y Arch Linux (requiere nfpm)

dist:
	@mkdir -p $(DIST)
	@set -e; for target in $(PLATFORMS); do \
		goos=$${target%%/*}; rest=$${target#*/}; goarch=$${rest%%/*}; \
		variant=""; case "$$rest" in */*) variant=$${rest#*/};; esac; \
		label=$$goarch; \
		if [ "$$goarch" = arm ]; then label=armv7; fi; \
		dir=$(DIST)/sessions-control_$(VERSION)_$${goos}_$$label; \
		if [ -z "$(FORCE)" ] && [ -f "$$dir.tar.gz" ]; then \
			echo ">> $$goos/$$goarch (existente)"; continue; \
		fi; \
		echo ">> $$goos/$$goarch$$variant"; \
		rm -rf "$$dir"; mkdir -p "$$dir/docs"; \
		CGO_ENABLED=0 GOOS=$$goos GOARCH=$$goarch GOARM=$${variant#v} \
			go build -trimpath -ldflags "$(LDFLAGS)" -o "$$dir/sessions-control" $(PKG); \
		cp README.md "$$dir/"; \
		cp docs/spec.md docs/AGENTS.md "$$dir/docs/"; \
		tar -C $(DIST) -czf "$$dir.tar.gz" "$$(basename $$dir)"; \
	done
	@$(MAKE) --no-print-directory checksums

checksums:
	@cd $(DIST) && find . -maxdepth 1 -type f \( \
		-name 'sessions-control_*.tar.gz' -o -name 'sessions-control*.deb' \
		-o -name 'sessions-control*.rpm' -o -name 'sessions-control*.apk' \
		-o -name 'sessions-control*.pkg.tar.zst' \) -printf '%f\n' \
		| sort | xargs -r sha256sum > checksums.txt
	@echo ">> $(DIST)/checksums.txt"

packages: deb rpm apk archlinux checksums

release:
	@for a in amd64 arm64 armv7 386; do \
		echo "== paquetes $$a"; \
		$(MAKE) --no-print-directory packages PKG_ARCH=$$a || exit 1; \
	done

deb rpm apk archlinux: %: dist
	@command -v $(NFPM) >/dev/null 2>&1 || { \
		echo ">> falta nfpm: go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest"; exit 1; }
	@test -f "$(DIST)/sessions-control_$(VERSION)_linux_$(PKG_ARCH)/sessions-control" || { \
		echo ">> compila primero: make dist (o revisa PKG_ARCH=$(PKG_ARCH))"; exit 1; }
	@arch="$(PKG_ARCH)"; case "$@" in \
	  deb) arch="$(PKG_ARCH_DEB)" ;; \
	  rpm) arch="$(PKG_ARCH_RPM)" ;; \
	  apk) arch="$(PKG_ARCH_APK)" ;; \
	  archlinux) arch="$(PKG_ARCH_ARCH)" ;; \
	esac; \
	sed -e 's|$${PKG_VERSION}|$(VERSION)|g' \
	    -e "s|\$${PKG_ARCH}|$$arch|g" \
	    -e 's|$${BINARY}|$(abspath $(DIST)/sessions-control_$(VERSION)_linux_$(PKG_ARCH)/sessions-control)|g' \
	    packaging/nfpm.yaml > $(DIST)/nfpm-$(PKG_ARCH).yaml
	$(NFPM) package -f $(DIST)/nfpm-$(PKG_ARCH).yaml -p $@ -t $(DIST)/

packaging-help:
	@echo "make dist                      tarballs linux/amd64 arm64 armv7 386"
	@echo "make deb|rpm|apk|archlinux     paquete nativo (PKG_ARCH=amd64 por defecto)"
	@echo "make packages                  los cuatro paquetes"
	@echo "make release                   paquetes para amd64/arm64/armv7/386"
	@echo "make dist FORCE=1              reconstruye tarballs"
	@echo "make install                   copia a $(PREFIX)"
	@echo "packaging/install.sh --user    instalador universal desde release"
