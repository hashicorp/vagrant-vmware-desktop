#!/usr/bin/env bash
set -e

extension="tar.xz"

# Get our directory
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"

. "${DIR}/common.sh"

echo "==> Installing PKGBUILD files..."

cp "${package}/xz/PKGBUILD" "${stage}/PKGBUILD"
cp "${package}/xz/vagrant-vmware-utility.install" "${stage}/vagrant-vmware-utility.install"
mv "${stage}/opt/vagrant-vmware-desktop/bin/vagrant-vmware-utility" "${stage}/vagrant-vmware-utility"

echo "==> Building ${extension} package..."

pushd "${stage}" > /dev/null

export version

makepkg --syncdeps --force --noconfirm
mv *.xz "${asset}"

popd > /dev/null

echo "==> Cleaning up packaging artifacts..."
rm -rf "${stage}"

echo "==> Package build complete: ${asset}"
