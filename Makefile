.PHONY: all test build run
SHELL := /bin/bash

include golang.mk
.DEFAULT_GOAL := test # override default goal set in library makefile

.PHONY: all test build clean
SHELL := /bin/bash
PKGS = $(shell GO15VENDOREXPERIMENT=1 go list ./... | grep -v "vendor/" | grep -v "db")
BINARY_NAME := "stealth"
$(eval $(call golang-version-check,1.8))

$(GOPATH)/bin/glide:
	@go get github.com/Masterminds/glide

all: build test

test: $(PKGS)
$(PKGS): golang-test-all-deps
	$(call golang-test-all,$@)

install_deps: $(GOPATH)/bin/glide
	@$(GOPATH)/bin/glide install

build:
	go build

run: build
	./stealth
