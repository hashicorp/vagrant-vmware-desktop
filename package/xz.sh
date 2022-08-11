#!/usr/bin/env bash

function fail() {
    echo "ERROR: ${1}"
    exit 1
}

extension="tar.xz"

# Get our directory
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"

. "${DIR}/common.sh"

echo "==> Installing PKGBUILD files..."

cp "${package}/xz/PKGBUILD" "${stage}/PKGBUILD" ||
    fail "Could not add PKGBUILD file"
cp "${package}/xz/vagrant-vmware-utility.install" "${stage}/vagrant-vmware-utility.install" ||
    fail "Could not add install script"
mv "${stage}/opt/vagrant-vmware-desktop/bin/vagrant-vmware-utility" "${stage}/vagrant-vmware-utility" ||
    fail "Could not add utility binary"

echo "==> Building ${extension} package..."

pushd "${stage}" > /dev/null 2>&1 ||
    fail "Could not enter staging directory"

export version

makepkg --syncdeps --force --noconfirm ||
    fail "Failed to make arch pacakge"

mv ./*.xz "${asset}"

popd > /dev/null 2>&1 || fail "Could not return to original directory"

echo "==> Cleaning up packaging artifacts..."
rm -rf "${stage}"

echo "==> Package build complete: ${asset}"
