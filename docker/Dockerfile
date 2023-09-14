FROM alpine

ARG VERSION=v1.11.0
ARG ARCH=linux
ARG CPU=amd64

RUN apk add wget \
  && echo "[*] Version: ${VERSION}" \
  && echo "[*] Arch: ${ARCH}" \
  && echo "[*] CPU: ${CPU}"

RUN wget \
    --quiet \
    -O dapr-cli.tgz \
    https://github.com/dapr/cli/releases/download/${VERSION}/dapr_${ARCH}_${CPU}.tar.gz \
  && tar -xf dapr-cli.tgz \
  && rm dapr-cli.tgz \
  && mv dapr /bin/dapr
