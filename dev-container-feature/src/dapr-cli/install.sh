#!/bin/sh
set -e


###################
# Helper Functions
###################

# Re-used from https://github.com/devcontainers/features/blob/4a9929f96485061e3778b35848e21d7c3c193480/src/dotnet/install.sh#L74

apt_get_update()
{
    if [ "$(find /var/lib/apt/lists/* | wc -l)" = "0" ]; then
        echo "Running apt-get update..."
        apt-get update -y
    fi
}

# Check if packages are installed and installs them if not.
check_packages() {
    if ! dpkg -s "$@" > /dev/null 2>&1; then
        apt_get_update
        apt-get -y install --no-install-recommends "$@"
    fi
}

get_latest_release() {
  curl --silent "https://api.github.com/repos/dapr/cli/releases/latest" |
  grep '"tag_name":' | sed -E "s/.*\"v([^\"]+)\".*/\1/"
}

###################
# Install Dapr CLI
###################
echo "Activating feature 'dapr-cli'"

check_packages curl ca-certificates

VERSION=${VERSION:-"latest"}

if [ "${VERSION}" = "latest" ]; then
  VERSION=$(get_latest_release)
fi

ARCH=$(uname -m)
case $ARCH in
    armv7*) ARCH="arm";;
    aarch64) ARCH="arm64";;
    x86_64) ARCH="amd64";;
esac

curl -SsL "https://github.com/dapr/cli/releases/download/v${VERSION}/dapr_linux_${ARCH}.tar.gz" | \
     tar -zx -C /usr/local/bin dapr

dapr --version

## Write bash completion code to a file and source it from .bash_profile
mkdir -p $_REMOTE_USER_HOME/.dapr
dapr completion bash >  $_REMOTE_USER_HOME/.dapr/completion.bash.inc
printf "
## dapr shell completion
source '$_REMOTE_USER_HOME/.dapr/completion.bash.inc'
" >> $_REMOTE_USER_HOME/.bashrc
chown -R $_REMOTE_USER:$_REMOTE_USER $_REMOTE_USER_HOME/.dapr
chown $_REMOTE_USER:$_REMOTE_USER $_REMOTE_USER_HOME/.bashrc
