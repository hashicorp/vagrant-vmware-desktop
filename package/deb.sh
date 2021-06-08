#!/usr/bin/env bash
set -e

extension="deb"

# Get our directory
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"

. "${DIR}/common.sh"

echo "==> Building ${extension} package..."

fpm -p ${asset} \
    -n vagrant-vmware-utility \
    -v $version \
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
    .

echo "==> Cleaning up packaging artifacts..."
rm -rf "${stage}"

echo "==> Package build complete: ${asset}"
