#!/bin/sh
set -e

echo "Activating feature 'dapr-cli'"

get_latest_release() {
  curl --silent "https://api.github.com/repos/dapr/cli/releases/latest" |
  grep '"tag_name":' | sed -E "s/.*\"v([^\"]+)\".*/\1/"
}

VERSION=${VERSION:-"latest"}

if [ "${VERSION}" = "latest" ]; then
  VERSION=$(get_latest_release)
fi

curl -SsL https://github.com/dapr/cli/releases/download/v"${VERSION}"/dapr_linux_amd64.tar.gz | \
     sudo tar -zx -C /usr/local/bin dapr

dapr --version

## Write bash completion code to a file and source if from .bash_profile
mkdir -p $_REMOTE_USER_HOME/.dapr
dapr completion bash >  $_REMOTE_USER_HOME/.dapr/completion.bash.inc
printf "
## dapr shell completion
source '$_REMOTE_USER_HOME/.dapr/completion.bash.inc'
" >> $_REMOTE_USER_HOME/.bashrc
sudo chown -R $_REMOTE_USER:$_REMOTE_USER $_REMOTE_USER_HOME/.dapr
sudo chown $_REMOTE_USER:$_REMOTE_USER $_REMOTE_USER_HOME/.bashrc