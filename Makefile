.PHONY: all test clean man fast release install version 

GO15VENDOREXPERIMENT=1

PROG_NAME := "sift"

# dirs
DIST_DIR ?= ./dist
WRK_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

# vcs
GIT_BRANCH := $(subst heads/,,$(shell git rev-parse --abbrev-ref HEAD 2>/dev/null))

# pkgs
SRCS = $(shell git ls-files '*.go' | grep -v '^vendor/')
PACKAGES = $(shell go list ./... | grep -v /vendor/)

# VERSION = $(shell cat "$(WRK_DIR)/VERSION" | tr '\n' '')
VERSION ?= $(shell git describe --tags)
VERSION_INCODE = $(shell perl -ne '/^var version.*"([^"]+)".*$$/ && print "v$$1\n"' main.go)
VERSION_INCHANGELOG = $(shell perl -ne '/^\# Release (\d+(\.\d+)+) / && print "$$1\n"' CHANGELOG.md | head -n1)
VERSION_INFILE := $(shell cat $(CURDIR)/VERSION)

VCS_GIT_REMOTE_URL = $(shell git config --get remote.origin.url)
VCS_GIT_VERSION ?= $(VERSION)

CURBIN := $(shell which $(PROG_NAME))

all: deps test build install version

build: semver
	@go build -ldflags "-X main.VERSION=`cat VERSION`" -o ./bin/$(PROG_NAME) ./cmd/$(PROG_NAME)/*.go
	@./bin/$(PROG_NAME) --version

semver:
	@echo "Previous version: $(VERSION_INFILE)"
	@echo "$(VERSION)" > $(CURDIR)/VERSION
	@echo "Current version: $(VERSION)"
	@echo "Current Install: $(CURBIN)"

install: semver deps
	@go install -ldflags "-X main.VERSION=`cat VERSION`" ./cmd/$(PROG_NAME)
	@$(PROG_NAME) --version

fast: deps
	@go build -i -ldflags "-X main.VERSION=`cat VERSION`-dev" -o ./bin/$(PROG_NAME) ./cmd/$(PROG_NAME)/*.go
	@$(PROG_NAME) --version

deps:
	@go get -v -u github.com/mattn/goveralls
	@go get -v -u golang.org/x/tools/cmd/cover
	@glide install --strip-vendor

ci: tests coverage

tests:
	@go test ./...

coverage:
	@go test -c -covermode=count -coverpkg=github.com/sniperkit/gotests/pkg,github.com/sniperkit/gotests/pkg/input,github.com/sniperkit/gotests/pkg/render,github.com/sniperkit/gotests/pkg/goparser,github.com/sniperkit/gotests/pkg/output,github.com/sniperkit/gotests/pkg/models
	@./gotests.test -test.coverprofile coverage.cov
	@goveralls -service=travis-ci -coverprofile=coverage.cov

clean:
	@go clean
	@rm -fr ./bin
	@rm -fr ./dist

release: $(PROG_NAME)
	@git tag -a `cat VERSION`
	@git push origin `cat VERSION`

man:
	@ronn man/textql.1.ronn
