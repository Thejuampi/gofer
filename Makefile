GO ?= go
VERSION ?= $(strip $(file < VERSION))

.PHONY: help build test fmt vet tidy install clean release

help:
	@echo Available targets:
	@echo   make build    Build the gofer binary
	@echo   make test     Run all tests
	@echo   make fmt      Format Go source files
	@echo   make vet      Run go vet
	@echo   make tidy     Run go mod tidy
	@echo   make install  Install gofer to GOPATH/bin
	@echo   make clean    Clean build and test caches
	@echo   make release  Vet + test + build \(release gate\)

build:
	$(GO) build -o gofer .

test:
	$(GO) test -count=1 -timeout 120s .

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

tidy:
	$(GO) mod tidy

install:
	$(GO) install .

clean:
	$(GO) clean -cache -testcache
	$(GO) clean .
	-rm -f gofer gofer.exe

release: vet test build
