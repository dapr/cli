#!/usr/bin/env bash

# ------------------------------------------------------------
# Copyright (c) Microsoft Corporation and Dapr Contributors.
# Licensed under the MIT License.
# ------------------------------------------------------------

# Dapr CLI location
: ${DAPR_INSTALL_DIR:="/usr/local/bin"}

# sudo is required to copy binary to DAPR_INSTALL_DIR for linux
: ${USE_SUDO:="false"}

# Http request CLI
DAPR_HTTP_REQUEST_CLI=curl

# GitHub Organization and repo name to download release
GITHUB_ORG=dapr
GITHUB_REPO=cli

# Dapr CLI filename
DAPR_CLI_FILENAME=dapr

DAPR_CLI_FILE="${DAPR_INSTALL_DIR}/${DAPR_CLI_FILENAME}"

getSystemInfo() {
    ARCH=$(uname -m)
    case $ARCH in
        armv7*) ARCH="arm";;
        aarch64) ARCH="arm64";;
        x86_64) ARCH="amd64";;
    esac

    OS=$(echo `uname`|tr '[:upper:]' '[:lower:]')

    # Most linux distro needs root permission to copy the file to /usr/local/bin
    if [ "$OS" == "linux" ] && [ "$DAPR_INSTALL_DIR" == "/usr/local/bin" ]; then
        USE_SUDO="true"
    fi
}

verifySupported() {
    local supported=(darwin-amd64 linux-amd64 linux-arm linux-arm64)
    local current_osarch="${OS}-${ARCH}"

    for osarch in "${supported[@]}"; do
        if [ "$osarch" == "$current_osarch" ]; then
            echo "Your system is ${OS}_${ARCH}"
            return
        fi
    done

    if [ "$current_osarch" == "darwin-arm64" ]; then
        echo "The darwin_arm64 arch has no native binary, however you can use the amd64 version so long as you have rosetta installed"
        echo "Use 'softwareupdate --install-rosetta' to install rosetta if you don't already have it"
        ARCH="amd64"
        return
    fi


    echo "No prebuilt binary for ${current_osarch}"
    exit 1
}

runAsRoot() {
    local CMD="$*"

    if [ $EUID -ne 0 -a $USE_SUDO = "true" ]; then
        CMD="sudo $CMD"
    fi

    $CMD
}

checkHttpRequestCLI() {
    if type "curl" > /dev/null; then
        DAPR_HTTP_REQUEST_CLI=curl
    elif type "wget" > /dev/null; then
        DAPR_HTTP_REQUEST_CLI=wget
    else
        echo "Either curl or wget is required"
        exit 1
    fi
}

checkExistingDapr() {
    if [ -f "$DAPR_CLI_FILE" ]; then
        echo -e "\nDapr CLI is detected:"
        $DAPR_CLI_FILE --version
        echo -e "Reinstalling Dapr CLI - ${DAPR_CLI_FILE}...\n"
    else
        echo -e "Installing Dapr CLI...\n"
    fi
}

getLatestRelease() {
    local daprReleaseUrl="https://api.github.com/repos/${GITHUB_ORG}/${GITHUB_REPO}/releases"
    local latest_release=""

    if [ "$DAPR_HTTP_REQUEST_CLI" == "curl" ]; then
        latest_release=$(curl -s $daprReleaseUrl | grep \"tag_name\" | grep -v rc | awk 'NR==1{print $2}' |  sed -n 's/\"\(.*\)\",/\1/p')
    else
        latest_release=$(wget -q --header="Accept: application/json" -O - $daprReleaseUrl | grep \"tag_name\" | grep -v rc | awk 'NR==1{print $2}' |  sed -n 's/\"\(.*\)\",/\1/p')
    fi

    ret_val=$latest_release
}

downloadFile() {
    LATEST_RELEASE_TAG=$1

    DAPR_CLI_ARTIFACT="${DAPR_CLI_FILENAME}_${OS}_${ARCH}.tar.gz"
    DOWNLOAD_BASE="https://github.com/${GITHUB_ORG}/${GITHUB_REPO}/releases/download"
    DOWNLOAD_URL="${DOWNLOAD_BASE}/${LATEST_RELEASE_TAG}/${DAPR_CLI_ARTIFACT}"

    # Create the temp directory
    DAPR_TMP_ROOT=$(mktemp -dt dapr-install-XXXXXX)
    ARTIFACT_TMP_FILE="$DAPR_TMP_ROOT/$DAPR_CLI_ARTIFACT"

    echo "Downloading $DOWNLOAD_URL ..."
    if [ "$DAPR_HTTP_REQUEST_CLI" == "curl" ]; then
        curl -SsL "$DOWNLOAD_URL" -o "$ARTIFACT_TMP_FILE"
    else
        wget -q -O "$ARTIFACT_TMP_FILE" "$DOWNLOAD_URL"
    fi

    if [ ! -f "$ARTIFACT_TMP_FILE" ]; then
        echo "failed to download $DOWNLOAD_URL ..."
        exit 1
    fi
}

installFile() {
    tar xf "$ARTIFACT_TMP_FILE" -C "$DAPR_TMP_ROOT"
    local tmp_root_dapr_cli="$DAPR_TMP_ROOT/$DAPR_CLI_FILENAME"

    if [ ! -f "$tmp_root_dapr_cli" ]; then
        echo "Failed to unpack Dapr CLI executable."
        exit 1
    fi

    chmod o+x $tmp_root_dapr_cli
    runAsRoot cp "$tmp_root_dapr_cli" "$DAPR_INSTALL_DIR"

    if [ -f "$DAPR_CLI_FILE" ]; then
        echo "$DAPR_CLI_FILENAME installed into $DAPR_INSTALL_DIR successfully."

        $DAPR_CLI_FILE --version
    else 
        echo "Failed to install $DAPR_CLI_FILENAME"
        exit 1
    fi
}

fail_trap() {
    result=$?
    if [ "$result" != "0" ]; then
        echo "Failed to install Dapr CLI"
        echo "For support, go to https://dapr.io"
    fi
    cleanup
    exit $result
}

cleanup() {
    if [[ -d "${DAPR_TMP_ROOT:-}" ]]; then
        rm -rf "$DAPR_TMP_ROOT"
    fi
}

installCompleted() {
    echo -e "\nTo get started with Dapr, please visit https://docs.dapr.io/getting-started/"
}

# -----------------------------------------------------------------------------
# main
# -----------------------------------------------------------------------------
trap "fail_trap" EXIT

getSystemInfo
verifySupported
checkExistingDapr
checkHttpRequestCLI


if [ -z "$1" ]; then
    echo "Getting the latest Dapr CLI..."
    getLatestRelease
else
    ret_val=v$1
fi

echo "Installing $ret_val Dapr CLI..."

downloadFile $ret_val
installFile
cleanup

installCompleted
