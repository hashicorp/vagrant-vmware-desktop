#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0


function fail() {
    echo "ERROR: ${1}"
    exit 1
}

# Get our directory
csource="${BASH_SOURCE[0]}"
while [ -h "$csource" ] ; do csource="$(readlink "$csource")"; done
package="$( cd -P "$( dirname "$csource" )" && pwd )"

if [[ "${extension}" == "" ]]; then
    fail "extension variable is unset!"
fi

root="$(dirname "${package}")"
base="${root}/pkg"
stage="${base}/${extension}"
binary_storage="${base}/binaries"

if [ "${extension}" = "dmg" ]; then
    binary_path="${binary_storage}/vagrant-vmware-utility_darwin_amd64"
else
    binary_path="${binary_storage}/vagrant-vmware-utility_linux_amd64"
fi

if [ ! -f "${binary_path}" ]; then
    fail "No utility binary found. Please run ${package}/init.sh"
fi

echo "==> Setting up for ${extension} build..."

if [ -d "${stage}" ]; then
    echo "    !! Removing existing ${stage} directory"
    rm -rf "${stage}"
fi

mkdir -p "${stage}/opt/vagrant-vmware-desktop/bin"

echo "==> Installing vagrant-vmware-utility..."

cp -f "${binary_path}" "${stage}/opt/vagrant-vmware-desktop/bin/vagrant-vmware-utility" ||
    fail "Could not add utility binary"
chmod -R 755 "${stage}/opt/vagrant-vmware-desktop" ||
    fail "Could not modify permissions on utility binary"

if [ -n "${UTILITY_VERSION}" ]; then
    version="${UTILITY_VERSION}"
else
    echo -n "==> Detecting utility version... "
    if ! version=$("${stage}/opt/vagrant-vmware-desktop/bin/vagrant-vmware-utility" -v 2>&1); then
        fail "Failed to read utility version"
    fi
    echo "${version}!"
fi

echo "==> Generating configuration and init files..."

mkdir -p "${stage}/opt/vagrant-vmware-desktop/config"
mkdir -p "${base}/init"

if [[ "${extension}" == "rpm" || "${extension}" == "deb" ]]; then
    # Generate sysv file
    "${stage}/opt/vagrant-vmware-desktop/bin/vagrant-vmware-utility" \
        service install \
        -print \
        -config-write "${stage}/opt/vagrant-vmware-desktop/config/service.hcl" \
        -config-path "/opt/vagrant-vmware-desktop/config/service.hcl" \
        -exe-path "/opt/vagrant-vmware-desktop/bin/vagrant-vmware-utility" \
        -init-style "sysv" > "${base}/init/vagrant-vmware-utility.init" ||
        fail "Failed to generate sysv init file"

    # Generate systemd file
    "${stage}/opt/vagrant-vmware-desktop/bin/vagrant-vmware-utility" \
        service install \
        -print \
        -config-write "${stage}/opt/vagrant-vmware-desktop/config/service.hcl" \
        -config-path "/opt/vagrant-vmware-desktop/config/service.hcl" \
        -exe-path "/opt/vagrant-vmware-desktop/bin/vagrant-vmware-utility" \
        -init-style "systemd" > "${base}/init/vagrant-vmware-utility.service" ||
        fail "Failed to generate systemd unit file"
fi

if [ -z "${RELEASE_NUMBER}" ]; then
    RELEASE_NUMBER="1"
fi

case "${extension}" in
    zst)
        export asset="${base}/vagrant-vmware-utility_${version}-${RELEASE_NUMBER}-x86_64.pkg.tar.zst";;
    rpm)
        export asset="${base}/vagrant-vmware-utility-${version}-${RELEASE_NUMBER}.x86_64.rpm";;
    deb)
        export asset="${base}/vagrant-vmware-utility_${version}-${RELEASE_NUMBER}_amd64.deb";;
    dmg)
        export asset="${base}/vagrant-vmware-utility_${version}_darwin_amd64.dmg";;
    zip)
        export asset="${base}/vagrant-vmware-utility_${version}_linux_amd64.zip";;
    msi)
        export asset="${base}/vagrant-vmware-utility_${version}_windows_amd64.msi";;
    *)
        fail "Unknown extension provided: ${extension}";;
esac
