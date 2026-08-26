APP := plane
DIST := dist
GO := go
GOFLAGS := -trimpath
LDFLAGS := -s -w
GOCACHE ?= /tmp/my-plane-go-cache
GOMODCACHE ?= /tmp/my-plane-go-mod-cache

.PHONY: build test skill release

build:
	mkdir -p $(DIST)
	GOCACHE="$(GOCACHE)" GOMODCACHE="$(GOMODCACHE)" CGO_ENABLED=0 $(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(DIST)/$(APP) ./cmd/plane

test:
	@files="$$(gofmt -l cmd internal)"; test -z "$$files" || (echo "gofmt required:"; echo "$$files"; exit 1)
	GOCACHE="$(GOCACHE)" GOMODCACHE="$(GOMODCACHE)" $(GO) test ./...
	GOCACHE="$(GOCACHE)" GOMODCACHE="$(GOMODCACHE)" $(GO) vet ./...

skill:
	rm -rf $(DIST)/skill-stage $(DIST)/my-plane-skill.zip
	mkdir -p $(DIST)/skill-stage/references
	cp SKILL.md $(DIST)/skill-stage/SKILL.md
	cp references/work-item-description.md $(DIST)/skill-stage/references/work-item-description.md
	(cd $(DIST)/skill-stage && zip -X -qr ../my-plane-skill.zip SKILL.md references)
	! unzip -Z1 $(DIST)/my-plane-skill.zip | grep -Eq '(^|/)scripts/plane(\.exe)?$$'

release: test skill
	rm -f $(DIST)/$(APP)-linux-amd64 $(DIST)/$(APP)-linux-arm64 $(DIST)/$(APP)-darwin-amd64 $(DIST)/$(APP)-darwin-arm64 $(DIST)/$(APP)-windows-amd64.exe $(DIST)/SHA256SUMS
	GOCACHE="$(GOCACHE)" GOMODCACHE="$(GOMODCACHE)" CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(DIST)/$(APP)-linux-amd64 ./cmd/plane
	GOCACHE="$(GOCACHE)" GOMODCACHE="$(GOMODCACHE)" CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(DIST)/$(APP)-linux-arm64 ./cmd/plane
	GOCACHE="$(GOCACHE)" GOMODCACHE="$(GOMODCACHE)" CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(DIST)/$(APP)-darwin-amd64 ./cmd/plane
	GOCACHE="$(GOCACHE)" GOMODCACHE="$(GOMODCACHE)" CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(DIST)/$(APP)-darwin-arm64 ./cmd/plane
	GOCACHE="$(GOCACHE)" GOMODCACHE="$(GOMODCACHE)" CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(DIST)/$(APP)-windows-amd64.exe ./cmd/plane
	@if command -v sha256sum >/dev/null 2>&1; then (cd $(DIST) && sha256sum $(APP)-* my-plane-skill.zip > SHA256SUMS); else (cd $(DIST) && shasum -a 256 $(APP)-* my-plane-skill.zip > SHA256SUMS); fi
