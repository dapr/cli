#!/usr/bin/env bash

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

# getSystemInfo discovers the architecture and OS for this system.
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


# verifySupported checks that the os/arch combination is supported for
# binary builds.
verifySupported() {
    local supported=(darwin-amd64 linux-amd64 linux-arm linux-arm64)
    local current_osarch="${OS}-${ARCH}"

    for osarch in "${supported[@]}"; do
        if [ "$osarch" == "$current_osarch" ]; then
            echo "Your system is ${OS}_${ARCH}"
            return
        fi
    done

    echo "No prebuilt binary for ${current_osarch}"
    exit 1
}

# runs the given command as root (detects if we are root already)
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
        latest_release=$(curl -s $daprReleaseUrl | grep \"tag_name\" | awk 'NR==1{print $2}' |  sed -n 's/\"\(.*\)\",/\1/p')
    else
        latest_release=$(wget -q --header="Accept: application/json" -O - $daprReleaseUrl | grep \"tag_name\" | awk 'NR==1{print $2}' |  sed -n 's/\"\(.*\)\",/\1/p')
    fi

    ret_val=$latest_release
}

# downloadFile downloads the latest binary
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

# installFile unpacks and installs CLI.
installFile() {
    tar xf "$ARTIFACT_TMP_FILE" -C "$DAPR_TMP_ROOT"
    chmod o+x "$DAPR_TMP_ROOT/$DAPR_CLI_FILENAME"
    runAsRoot cp "$DAPR_TMP_ROOT/$DAPR_CLI_FILENAME" "$DAPR_INSTALL_DIR"

    if [ -f "$DAPR_CLI_FILE" ]; then
        echo "$DAPR_CLI_FILENAME installed into $DAPR_INSTALL_DIR successfully."

        $DAPR_CLI_FILE --version
    fi
}

# fail_trap is executed if an error occurs.
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
    echo -e "\nTo get started with Dapr, please visit https://github.com/dapr/docs/tree/master/getting-started"
}

# -----------------------------------------------------------------------------
# main
# -----------------------------------------------------------------------------
trap "fail_trap" EXIT

getSystemInfo
verifySupported
checkExistingDapr
checkHttpRequestCLI

getLatestRelease
downloadFile $ret_val
installFile
cleanup

installCompleted
