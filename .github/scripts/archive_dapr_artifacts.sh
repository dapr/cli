#!/usr/bin/env bash

# ------------------------------------------------------------
# Copyright 2021 The Dapr Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file_base except in compliance with the License.
# You may obtain a copy of the License at
#     http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
# ------------------------------------------------------------

# Http request CLI
DAPR_HTTP_REQUEST_CLI=curl

# GitHub Organization and repo name to download release
GITHUB_ORG=dapr
GITHUB_DAPR_REPO=dapr
GITHUB_DASHBOARD_REPO=dashboard

# Dapr binaries filename
DAPRD_FILENAME=daprd
PLACEMENT_FILENAME=placement
DASHBOARD_FILENAME=dashboard

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

getLatestRelease() {
    local daprReleaseUrl="https://api.github.com/repos/${GITHUB_ORG}/$1/releases"
    local latest_release=""

    if [ "$DAPR_HTTP_REQUEST_CLI" == "curl" ]; then
        latest_release=$(curl -s $daprReleaseUrl | grep \"tag_name\" | grep -v rc | awk 'NR==1{print $2}' |  sed -n 's/\"\(.*\)\",/\1/p')
    else
        latest_release=$(wget -q --header="Accept: application/json" -O - $daprReleaseUrl | grep \"tag_name\" | grep -v rc | awk 'NR==1{print $2}' |  sed -n 's/\"\(.*\)\",/\1/p')
    fi

    echo $latest_release
}

downloadFile() {
  local repo=$1
  local file_base=$2
  local ver=$3
  local platform=$4
  local ext="tar.gz"
  local out_dir=${TARGET_DIR}/dist

  if [[ "${platform}" == "windows_amd64" ]]; then
    ext="zip"
  fi

  local filename=${file_base}_${platform}.${ext}
  local download_url="https://github.com/${GITHUB_ORG}/${repo}/releases/download/${ver}/${filename}"

  mkdir -p ${out_dir}

  echo "Downloading $download_url to ${out_dir}/${filename}..."
  if [ "$DAPR_HTTP_REQUEST_CLI" == "curl" ]; then
    curl -SsL "$download_url" -o "${out_dir}/${filename}"
  else
    wget -q -O "${out_dir}/${filename}" "$download_url"
  fi

  if [ ! -f "${out_dir}/${filename}" ]; then
    echo "failed to download $download_url ..."
    exit 1
  fi
}

downloadDockerImage() {
  local image_name=$1
  local image_ver=$2
  local docker_image="${image_name}:${image_ver}"
  if [[ ${image_ver} == "latest" ]]; then
    docker_image=${image_name}
  fi
  local file_name=$(echo ${docker_image} | sed -e 's/\//-/g' | sed -e 's/:/-/g').tar.gz
  local out_dir=${TARGET_DIR}/docker

  mkdir -p ${out_dir}

  echo "Pulling docker image ${docker_image}..."
  docker pull "${docker_image}"
  echo "Saving docker image to ${out_dir}/${file_name}..."
  docker save -o "${out_dir}/${file_name}" "${docker_image}"
}

downloadArtifacts() {
  for platform in "darwin_amd64" "darwin_arm64" "linux_amd64" "linux_arm64" "windows_amd64"; do
    downloadFile ${GITHUB_DAPR_REPO} ${DAPRD_FILENAME} ${RUNTIME_VER} ${platform}
    downloadFile ${GITHUB_DAPR_REPO} ${PLACEMENT_FILENAME} ${RUNTIME_VER} ${platform}
    downloadFile ${GITHUB_DASHBOARD_REPO} ${DASHBOARD_FILENAME} ${DASHBOARD_VER} ${platform}
  done
}

downloadDockerImages() {
  downloadDockerImage "daprio/dapr" $(echo ${RUNTIME_VER} | sed -e 's/^v//')
  downloadDockerImage "openzipkin/zipkin" "latest"
  downloadDockerImage "redis" "latest"
}

writeVersion() {
  local runtime_ver=$(echo ${RUNTIME_VER} | sed -e 's/^v//')
  local dashboard_ver=$(echo ${DASHBOARD_VER} | sed -e 's/^v//')
  cat <<VERSION > ${TARGET_DIR}/version.json
{
  "daprd": "${runtime_ver}",
  "dashboard": "${dashboard_ver}"
}
VERSION
}

fail_trap() {
    result=$?
    if [ "$result" != "0" ]; then
        echo "Failed to download Dapr artifacts"
        echo "For support, go to https://dapr.io"
    fi
    exit $result
}

# -----------------------------------------------------------------------------
# main
# -----------------------------------------------------------------------------
trap "fail_trap" EXIT

while getopts 'r:d:o:' options; do
  case ${options} in
    r) RUNTIME_VER=${OPTARG};;
    d) DASHBOARD_VER=${OPTARG};;
    o) TARGET_DIR=${OPTARG};;
  esac
done

checkHttpRequestCLI

if [[ -z ${RUNTIME_VER} ]]; then
  RUNTIME_VER=$(getLatestRelease ${GITHUB_DAPR_REPO})
fi

if [[ -z ${DASHBOARD_VER} ]]; then
  DASHBOARD_VER=$(getLatestRelease ${GITHUB_DASHBOARD_REPO})
fi

if [[ -z ${TARGET_DIR} ]]; then
  TARGET_DIR=artifacts
fi

mkdir -p ${TARGET_DIR}
writeVersion
downloadArtifacts
downloadDockerImages


