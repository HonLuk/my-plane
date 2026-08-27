APP := plane
SKILL_NAME := my-plane-skill
DIST := dist
GO := go
GOFLAGS := -trimpath
LDFLAGS := -s -w
GOCACHE ?= /tmp/my-plane-go-cache
GOMODCACHE ?= /tmp/my-plane-go-mod-cache

RELEASE_TARGETS := linux-amd64 linux-arm64 darwin-amd64 darwin-arm64 windows-amd64
RELEASE_BINARY_NAMES := \
	$(APP)-linux-amd64 \
	$(APP)-linux-arm64 \
	$(APP)-darwin-amd64 \
	$(APP)-darwin-arm64 \
	$(APP)-windows-amd64.exe
RELEASE_SKILL_ARCHIVE_NAMES := \
	$(SKILL_NAME)-linux-amd64.zip \
	$(SKILL_NAME)-linux-arm64.zip \
	$(SKILL_NAME)-darwin-amd64.zip \
	$(SKILL_NAME)-darwin-arm64.zip \
	$(SKILL_NAME)-windows-amd64.zip
RELEASE_ASSET_NAMES := $(RELEASE_BINARY_NAMES) $(RELEASE_SKILL_ARCHIVE_NAMES)
RELEASE_BINARIES := $(addprefix $(DIST)/,$(RELEASE_BINARY_NAMES))
RELEASE_SKILL_ARCHIVES := $(addprefix $(DIST)/,$(RELEASE_SKILL_ARCHIVE_NAMES))

.PHONY: build test release-binaries skill release

build:
	mkdir -p $(DIST)
	GOCACHE="$(GOCACHE)" GOMODCACHE="$(GOMODCACHE)" CGO_ENABLED=0 $(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(DIST)/$(APP) ./cmd/plane

test:
	@files="$$(gofmt -l cmd internal)"; test -z "$$files" || (echo "gofmt required:"; echo "$$files"; exit 1)
	GOCACHE="$(GOCACHE)" GOMODCACHE="$(GOMODCACHE)" $(GO) test ./...
	GOCACHE="$(GOCACHE)" GOMODCACHE="$(GOMODCACHE)" $(GO) vet ./...

release-binaries:
	mkdir -p $(DIST)
	rm -f $(RELEASE_BINARIES)
	GOCACHE="$(GOCACHE)" GOMODCACHE="$(GOMODCACHE)" CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(DIST)/$(APP)-linux-amd64 ./cmd/plane
	GOCACHE="$(GOCACHE)" GOMODCACHE="$(GOMODCACHE)" CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(DIST)/$(APP)-linux-arm64 ./cmd/plane
	GOCACHE="$(GOCACHE)" GOMODCACHE="$(GOMODCACHE)" CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(DIST)/$(APP)-darwin-amd64 ./cmd/plane
	GOCACHE="$(GOCACHE)" GOMODCACHE="$(GOMODCACHE)" CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(DIST)/$(APP)-darwin-arm64 ./cmd/plane
	GOCACHE="$(GOCACHE)" GOMODCACHE="$(GOMODCACHE)" CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(DIST)/$(APP)-windows-amd64.exe ./cmd/plane

skill: release-binaries
	rm -rf $(DIST)/skill-stage
	rm -f $(RELEASE_SKILL_ARCHIVES) $(DIST)/$(SKILL_NAME).zip
	mkdir -p $(DIST)/skill-stage/references
	mkdir -p $(DIST)/skill-stage/scripts
	cp SKILL.md $(DIST)/skill-stage/SKILL.md
	cp references/work-item-description.md $(DIST)/skill-stage/references/work-item-description.md
	@set -e; \
	for target in $(RELEASE_TARGETS); do \
		case "$$target" in \
			windows-amd64) binary="$(DIST)/$(APP)-windows-amd64.exe"; packaged="plane.exe" ;; \
			*) binary="$(DIST)/$(APP)-$$target"; packaged="plane" ;; \
		esac; \
		rm -f "$(DIST)/skill-stage/scripts/plane" "$(DIST)/skill-stage/scripts/plane.exe"; \
		cp "$$binary" "$(DIST)/skill-stage/scripts/$$packaged"; \
		chmod +x "$(DIST)/skill-stage/scripts/$$packaged"; \
		(cd $(DIST)/skill-stage && zip -X -qr "../$(SKILL_NAME)-$$target.zip" SKILL.md references scripts); \
	done
	rm -rf $(DIST)/skill-stage
	@set -e; \
	for target in $(RELEASE_TARGETS); do \
		archive="$(DIST)/$(SKILL_NAME)-$$target.zip"; \
		unzip -Z1 "$$archive" | grep -Fx 'SKILL.md' >/dev/null; \
		unzip -Z1 "$$archive" | grep -Fx 'references/work-item-description.md' >/dev/null; \
		if [ "$$target" = windows-amd64 ]; then packaged='scripts/plane.exe'; else packaged='scripts/plane'; fi; \
		unzip -Z1 "$$archive" | grep -Fx "$$packaged" >/dev/null; \
	done

release: test skill
	rm -f $(DIST)/SHA256SUMS
	@if command -v sha256sum >/dev/null 2>&1; then (cd $(DIST) && sha256sum $(RELEASE_ASSET_NAMES) > SHA256SUMS); else (cd $(DIST) && shasum -a 256 $(RELEASE_ASSET_NAMES) > SHA256SUMS); fi
