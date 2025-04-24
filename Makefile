.PHONY: all test build run install_deps

include golang.mk
.DEFAULT_GOAL := test # override default goal set in library makefile

SHELL := /bin/bash
APP_NAME ?= stealth
EXECUTABLE = $(APP_NAME)
PKG = github.com/Clever/$(APP_NAME)
PKGS = $(shell go list ./... | grep -v /gen-go | grep -v /tools)
$(eval $(call golang-version-check,1.24))

all: test build

test: $(PKGS)
$(PKGS): golang-test-all-deps
	$(call golang-test-all,$@)

build:
	go build

run: build
	./stealth

install_deps: vendor
vendor: go.mod go.sum
	go mod vendor