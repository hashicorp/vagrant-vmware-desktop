#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0


function fail() {
    echo "ERROR: ${1}"
    exit 1
}

extension="deb"

# Get our directory
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"

. "${DIR}/common.sh"

echo "==> Building ${extension} package..."

fpm -p "${asset}" \
    -n vagrant-vmware-utility \
    -v "${version}" \
    -t deb \
    -s dir \
    -C "${stage}" \
    --log error \
    --prefix '/' \
    --maintainer "HashiCorp <support@hashicorp.com>" \
    --url "https://www.vagrantup.com/" \
    --epoch 1 \
    --deb-user root \
    --deb-group root \
    --deb-init "${root}/pkg/init/vagrant-vmware-utility.init" \
    --deb-systemd "${root}/pkg/init/vagrant-vmware-utility.service" \
    --deb-systemd-enable \
    --deb-systemd-auto-start \
    --after-install "${root}/package/common/after_install.sh" \
    . || fail "Failed to build deb package"

echo "==> Cleaning up packaging artifacts..."
rm -rf "${stage}"

echo "==> Package build complete: ${asset}"
