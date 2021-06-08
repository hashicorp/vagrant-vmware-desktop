#!/usr/bin/env bash
set -e

extension="rpm"

# Get our directory
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"

. "${DIR}/common.sh"

echo "==> Building ${extension} package... "

fpm -p ${asset} \
    -n vagrant-vmware-utility \
    -v $version \
    -t rpm \
    -s dir \
    -C "${stage}" \
    --log error \
    --prefix '/' \
    --maintainer "HashiCorp <support@hashicorp.com>" \
    --url "https://www.vagrantup.com/" \
    --epoch 1 \
    --license "MIT" \
    --description "Vagrant utility for VMware Workstation" \
    --config-files "opt/vagrant-vmware-desktop/config/service.hcl" \
    --rpm-init "${root}/pkg/init/vagrant-vmware-utility.init" \
    --rpm-auto-add-directories \
    --rpm-user root \
    --rpm-group root \
    --after-install "${root}/package/common/after_install.sh" \
    .

echo "==> Cleaning up packaging artifacts..."
rm -rf "${stage}"

echo "==> Package build complete: ${asset}"
