#!/bin/bash

# This test file will be executed against an auto-generated devcontainer.json that
# includes the 'dapr-cli' Feature with no options.
#
# For more information, see: https://github.com/devcontainers/cli/blob/main/docs/features/test.md
#
# Eg:
# {
#    "image": "<..some-base-image...>",
#    "features": {
#      "dapr-cli": {}
#    },
#    "remoteUser": "root"
# }
#
# Thus, the value of all options will fall back to the default value in 
# the Feature's 'devcontainer-feature.json'.
#
# These scripts are run as 'root' by default. Although that can be changed
# with the '--remote-user' flag.
# 
# This test can be run with the following command (from the repo root folder):
#
#    devcontainer features test \ 
#                   --features dapr-cli \
#                   --remote-user root  \
#                   --skip-scenarios    \
#                   --base-image mcr.microsoft.com/devcontainers/base:ubuntu \
#                   --project-folder dev-container-feature

set -e

# Optional: Import test library bundled with the devcontainer CLI
# Provides the 'check' and 'reportResults' commands.
source dev-container-features-test-lib

# Feature-specific tests
# The 'check' command comes from the dev-container-features-test-lib.
check "execute command" bash -c "dapr --help | grep 'Distributed Application Runtime'"

# Report results
# If any of the checks above exited with a non-zero exit code, the test will fail.
reportResults
