################################################################################
# Variables																       #
################################################################################

GIT_VERSION = $(shell git describe --always --abbrev=7 --dirty)
TARGETS		?= darwin windows linux
ARCH		?= amd64
CGO			?= 0

ifdef REL_VERSION
	CLI_VERSION := $(REL_VERSION)
else
	CLI_VERSION := edge
endif

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
# Build and release															   #
################################################################################

.PHONY: build
build:
	  for t in $(TARGETS); do \
				CGO_ENABLED=$(CGO) GOOS=$$t GOARCH=$(ARCH) go build \
						-ldflags "-X main.version=$(CLI_VERSION)" \
						-o dist/"$$t"_$(ARCH)/actions; \
	  done;


.PHONY: release
release: build
release: test
		cd dist; \
		for t in $(TARGETS); do \
				tar -zcf "$$t"_$(ARCH)/actions-v${CLI_VERSION}-$$t-$(ARCH).tar.gz "$$t"_$(ARCH)/* ; \
		done;

################################################################################
# Tests																           #
################################################################################
.PHONY: test
test:
	go test ./pkg/...
