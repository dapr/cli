# ------------------------------------------------------------
# Copyright (c) Microsoft Corporation and Dapr Contributors.
# Licensed under the MIT License.
# ------------------------------------------------------------

################################################################################
# Variables																       #
################################################################################

export GO111MODULE ?= on
export GOPROXY ?= https://proxy.golang.org
export GOSUMDB ?= sum.golang.org
GIT_COMMIT  = $(shell git rev-list -1 HEAD)
GIT_VERSION = $(shell git describe --always --abbrev=7 --dirty)
CGO			?= 0
CLI_BINARY  = dapr

ifdef REL_VERSION
	CLI_VERSION := $(REL_VERSION)
else
	CLI_VERSION := edge
endif

ifdef API_VERSION
	RUNTIME_API_VERSION = $(API_VERSION)
else
	RUNTIME_API_VERSION = 1.0
endif

LOCAL_ARCH := $(shell uname -m)
ifeq ($(LOCAL_ARCH),x86_64)
	TARGET_ARCH_LOCAL = amd64
else ifeq ($(shell echo $(LOCAL_ARCH) | head -c 5),armv8)
	TARGET_ARCH_LOCAL = arm64
else ifeq ($(shell echo $(LOCAL_ARCH) | head -c 5),aarch64)
	TARGET_ARCH_LOCAL = arm64
else ifeq ($(shell echo $(LOCAL_ARCH) | head -c 4),armv)
	TARGET_ARCH_LOCAL = arm
else
	TARGET_ARCH_LOCAL = amd64
endif
export GOARCH ?= $(TARGET_ARCH_LOCAL)

LOCAL_OS := $(shell uname)
ifeq ($(LOCAL_OS),Linux)
   TARGET_OS_LOCAL = linux
   GOLANGCI_LINT:=golangci-lint
   export ARCHIVE_EXT = .tar.gz
else ifeq ($(LOCAL_OS),Darwin)
   TARGET_OS_LOCAL = darwin
   GOLANGCI_LINT:=golangci-lint
   export ARCHIVE_EXT = .tar.gz
else
   TARGET_OS_LOCAL ?= windows
   BINARY_EXT_LOCAL = .exe
   GOLANGCI_LINT:=golangci-lint.exe
   export ARCHIVE_EXT = .zip
endif
export GOOS ?= $(TARGET_OS_LOCAL)
export BINARY_EXT ?= $(BINARY_EXT_LOCAL)

TEST_OUTPUT_FILE ?= test_output.json

# Use the variable H to add a header (equivalent to =>) to informational output
H = $(shell printf "\033[34;1m=>\033[0m")

ifeq ($(origin DEBUG), undefined)
  BUILDTYPE_DIR:=release
else ifeq ($(DEBUG),0)
  BUILDTYPE_DIR:=release
else
  BUILDTYPE_DIR:=debug
  GCFLAGS:=-gcflags="all=-N -l"
  $(info $(H) Build with debugger information)
endif

################################################################################
# Go build details                                                             #
################################################################################
BASE_PACKAGE_NAME := github.com/dapr/cli
OUT_DIR := ./dist

BINS_OUT_DIR := $(OUT_DIR)/$(GOOS)_$(GOARCH)/$(BUILDTYPE_DIR)
LDFLAGS := "-X main.version=$(CLI_VERSION) -X main.apiVersion=$(RUNTIME_API_VERSION)"

################################################################################
# Target: build                                                                #
################################################################################
.PHONY: build
build: $(CLI_BINARY)

$(CLI_BINARY):
	CGO_ENABLED=$(CGO) GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(GCFLAGS) -ldflags $(LDFLAGS) \
	-o $(BINS_OUT_DIR)/$(CLI_BINARY)$(BINARY_EXT);

################################################################################
# Target: lint                                                                 #
################################################################################
.PHONY: lint
lint:
	$(GOLANGCI_LINT) run --timeout=20m

################################################################################
# Target: archive                                                              #
################################################################################
ARCHIVE_OUT_DIR ?= $(BINS_OUT_DIR)

archive: archive-$(CLI_BINARY)$(ARCHIVE_EXT)

ifeq ($(GOOS),windows)
archive-$(CLI_BINARY).zip:
	7z.exe a -tzip "$(ARCHIVE_OUT_DIR)\\$(CLI_BINARY)_$(GOOS)_$(GOARCH)$(ARCHIVE_EXT)" "$(BINS_OUT_DIR)\\$(CLI_BINARY)$(BINARY_EXT)"
else
archive-$(CLI_BINARY).tar.gz:
	chmod +x $(BINS_OUT_DIR)/$(CLI_BINARY)$(BINARY_EXT)
	tar czf "$(ARCHIVE_OUT_DIR)/$(CLI_BINARY)_$(GOOS)_$(GOARCH)$(ARCHIVE_EXT)" -C "$(BINS_OUT_DIR)" "$(CLI_BINARY)$(BINARY_EXT)"
endif

################################################################################
# Target: release                                                              #
################################################################################
.PHONY: release
release: build archive

.PHONY: test-deps
test-deps:
	# The desire here is to download this test dependency without polluting go.mod
	# In golang >=1.16 there is a new way to do this with `go install gotest.tools/gotestsum@latest`
	# But this doesn't work with <=1.15, so we do it the old way for now 
	# (see: https://golang.org/ref/mod#go-install)
	GO111MODULE=off go get gotest.tools/gotestsum
################################################################################
# Tests																           #
################################################################################
.PHONY: test
test: test-deps
	gotestsum --jsonfile $(TEST_OUTPUT_FILE) --format standard-quiet -- ./pkg/... $(COVERAGE_OPTS)

################################################################################
# E2E Tests for Kubernetes												       #
################################################################################
.PHONY: test-e2e-k8s
test-e2e-k8s: test-deps
	gotestsum --jsonfile $(TEST_OUTPUT_FILE) --format standard-verbose -- -timeout 20m -count=1 -tags=e2e ./tests/e2e/kubernetes/... -run="TestKubernetes"

################################################################################
# Build, E2E Tests for Kubernetes											   #
################################################################################
.PHONY: e2e-build-run-k8s
e2e-build-run-k8s: build test-e2e-k8s

################################################################################
# E2E Tests for Kubernetes Upgrade											   #
################################################################################
.PHONY: test-e2e-upgrade
test-e2e-upgrade: test-deps
	gotestsum --jsonfile $(TEST_OUTPUT_FILE) --format standard-verbose -- -timeout 20m -count=1 -tags=e2e ./tests/e2e/kubernetes/... -run="TestUpgradePath"

################################################################################
# Build, E2E Tests for Kubernetes Upgrade									   #
################################################################################
.PHONY: e2e-build-run-upgrade
e2e-build-run-upgrade: build test-e2e-upgrade


################################################################################
# E2E Tests for Self-Hosted												       #
################################################################################
.PHONY: test-e2e-sh
test-e2e-sh: test-deps
	gotestsum --jsonfile $(TEST_OUTPUT_FILE) --format standard-verbose -- -count=1 -tags=e2e ./tests/e2e/standalone/...

################################################################################
# Build, E2E Tests for Self-Hosted											   #
################################################################################
.PHONY: e2e-build-run-sh
e2e-build-run-sh: build test-e2e-sh

################################################################################
# Target: go.mod                                                               #
################################################################################
.PHONY: go.mod
go.mod:
	go mod tidy

################################################################################
# Target: check-diff                                                           #
################################################################################
.PHONY: check-diff
check-diff:
	git diff --exit-code ./go.mod # check no changes
