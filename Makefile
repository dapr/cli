################################################################################
# Variables																       #
################################################################################

GIT_VERSION = $(shell git describe --always --abbrev=7 --dirty)
TARGETS		?= darwin linux windows
ARCH		?= amd64
CGO			?= 0

ifdef REL_VERSION
	CLI_VERSION := $(REL_VERSION)
else
	CLI_VERSION := edge
endif

################################################################################
# Go build details                                                             #
################################################################################

BASE_PACKAGE_NAME := github.com/actionscore/cli

################################################################################
# Dependencies																   #
################################################################################

.PHONY: dep
dep:
ifeq ($(shell command -v dep 2> /dev/null),)
	go get -u -v github.com/golang/dep/cmd/dep
endif

.PHONY: deps
deps: dep
	dep ensure -v

################################################################################
# Build																           #
################################################################################

.PHONY: build
build:
	  for t in $(TARGETS); do \
				CGO_ENABLED=$(CGO) GOOS=$$t GOARCH=$(ARCH) go build \
						-ldflags "-X $(BASE_PACKAGE_NAME)/pkg/version.version=$(CLI_VERSION)" \
						-o dist/"$$t"_$(ARCH)/actions; \
	  done;

################################################################################
# Tests																           #
################################################################################
.PHONY: test
test:
	go test ./pkg/...
