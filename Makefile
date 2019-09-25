################################################################################
# Variables																       #
################################################################################

export GO111MODULE ?= on
export GOPROXY ?= https://proxy.golang.org
export GOSUMDB ?= sum.golang.org
GIT_COMMIT  = $(shell git rev-list -1 HEAD)
GIT_VERSION = $(shell git describe --always --abbrev=7 --dirty)
CGO			?= 0
CLI_BINARY  = actions

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
else ifeq ($(shell echo $(LOCAL_ARCH) | head -c 4),armv)
	TARGET_ARCH_LOCAL = arm
else
	TARGET_ARCH_LOCAL = amd64
endif
export GOARCH ?= $(TARGET_ARCH_LOCAL)

LOCAL_OS := $(shell uname)
ifeq ($(LOCAL_OS),Linux)
   TARGET_OS_LOCAL = linux
   export ARCHIVE_EXT = .tar.gz
else ifeq ($(LOCAL_OS),Darwin)
   TARGET_OS_LOCAL = darwin
   export ARCHIVE_EXT = .tar.gz
else
   TARGET_OS_LOCAL ?= windows
   BINARY_EXT_LOCAL = .exe
   export ARCHIVE_EXT = .zip
endif
export GOOS ?= $(TARGET_OS_LOCAL)
export BINARY_EXT ?= $(BINARY_EXT_LOCAL)

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
BASE_PACKAGE_NAME := github.com/actionscore/cli
OUT_DIR := ./dist

BINS_OUT_DIR := $(OUT_DIR)/$(GOOS)_$(GOARCH)/$(BUILDTYPE_DIR)
LDFLAGS := "-X main.version=$(CLI_VERSION) -X main.apiVersion=$(RUNTIME_API_VERSION)"

################################################################################
# Target: build                                                                #
################################################################################
.PHONY: build
build: $(CLI_BINARY)

$(CLI_BINARY):
	CGO_ENABLED=$(CGO) GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(GCFLAGS) -ldflags $(LDFLAGS) -mod=vendor \
	-o $(BINS_OUT_DIR)/$(CLI_BINARY)$(BINARY_EXT);

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

################################################################################
# Tests																           #
################################################################################
.PHONY: test
test:
	go test ./pkg/... -mod=vendor
