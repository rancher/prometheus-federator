#!/bin/bash

set -e
set -x

YQ_VERSION=v4.44.3
YQ_CHECKSUM_AMD64='a2c097180dd884a8d50c956ee16a9cec070f30a7947cf4ebf87d5f36213e9ed7'
YQ_CHECKSUM_ARM64='0e7e1524f68d91b3ff9b089872d185940ab0fa020a5a9052046ef10547023156'

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)
    YQ_ARCH="amd64"
    YQ_CHECKSUM="$YQ_CHECKSUM_AMD64"
    ;;
  aarch64 | arm64)
    YQ_ARCH="arm64"
    YQ_CHECKSUM="$YQ_CHECKSUM_ARM64"
    ;;
  *)
    echo "Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

YQ_BINARY="yq_linux_${YQ_ARCH}"
YQ_URL="https://github.com/mikefarah/yq/releases/download/${YQ_VERSION}/${YQ_BINARY}"

# Download yq
echo "Downloading yq ${YQ_VERSION} for ${YQ_ARCH}..."
curl -fsSL "$YQ_URL" -o /tmp/yq

# Validate checksum
echo "Validating checksum..."
ACTUAL_CHECKSUM=$(sha256sum /tmp/yq | awk '{print $1}')

if [ "$ACTUAL_CHECKSUM" != "$YQ_CHECKSUM" ]; then
  echo "Checksum mismatch!" >&2
  echo "  Expected: $YQ_CHECKSUM" >&2
  echo "  Actual:   $ACTUAL_CHECKSUM" >&2
  rm -f /tmp/yq
  exit 1
fi

echo "Checksum OK."

# Install yq
chmod +x /tmp/yq
mv /tmp/yq /usr/local/bin/yq

echo "yq ${YQ_VERSION} installed successfully to /usr/local/bin/yq"
yq --version