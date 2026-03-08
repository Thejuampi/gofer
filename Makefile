GO ?= go
VERSION ?= $(strip $(file < VERSION))
STATICCHECK_VERSION ?= v0.7.0
INEFFASSIGN_VERSION ?= v0.2.0
ERRCHECK_VERSION ?= v1.10.0
GOVULNCHECK_VERSION ?= v1.1.4

.PHONY: help build test test-race fmt vet static-scan vuln-scan tidy install clean release

help:
	@echo Available targets:
	@echo   make build    Build the gofer binary
	@echo   make static-scan Run blocking static analysis
	@echo   make test     Run all tests
	@echo   make test-race Run all tests with the race detector
	@echo   make fmt      Format Go source files
	@echo   make vet      Run go vet
	@echo   make vuln-scan Run advisory vulnerability scan
	@echo   make tidy     Run go mod tidy
	@echo   make install  Install gofer to GOPATH/bin
	@echo   make clean    Clean build and test caches
	@echo   make release  Vet + test + build \(release gate\)

build:
	$(GO) build -o gofer .

test:
	$(GO) test -count=1 -timeout 120s .

test-race:
	$(GO) test -race -count=1 -timeout 180s .

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

static-scan:
	$(GO) vet ./...
	$(GO) run honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION) -checks=SA* ./...
	$(GO) run github.com/gordonklaus/ineffassign@$(INEFFASSIGN_VERSION) ./...
	$(GO) run github.com/kisielk/errcheck@$(ERRCHECK_VERSION) ./...

vuln-scan:
	$(GO) run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

tidy:
	$(GO) mod tidy

install:
	$(GO) install .

clean:
	$(GO) clean -cache -testcache
	$(GO) clean .
	-rm -f gofer gofer.exe

release: static-scan test-race test build
