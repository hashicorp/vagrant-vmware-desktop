#!/usr/bin/env bash

function fail() {
    echo "ERROR: ${1}"
    exit 1
}

win_sig_file="${WINDOWS_SIGNING_FILE:-./Win_CodeSigning.p12}"

# Get our directory
csource="${BASH_SOURCE[0]}"
while [ -h "$csource" ] ; do csource="$(readlink "$csource")"; done
package="$( cd -P "$( dirname "$csource" )" && pwd )"
root="$(dirname "${package}")"
vmware_utility="${root}/go_src/vagrant-vmware-utility"
binary_storage="${root}/pkg/binaries"

echo "==> Building Vagrant VMware utility..."

pushd "${vmware_utility}" > /dev/null 2>&1 ||
    fail "Could not enter utility source directory"

gox -os="linux darwin windows" -arch="amd64"

popd > /dev/null 2>&1 || fail "Failed to return to original directory"

if [ -f "${win_sig_file}" ]; then
    wfs=("${vmware_utility}/vagrant-vmware-utility_"*.exe)
    win_file="${wfs[0]}"
    if [ ! -f "${win_file}" ]; then
        fail "Could not locate windows utility binary"
    fi

    echo "==> Signing Windows binary (first pass)"
    osslsigncode sign -pkcs12 "${win_sig_file}" -pass "${SignKeyPassword}" -n "Vagrant VMware Utility" \
                 -i "https://www.vagrantup.com" -t "http://timestamp.digicert.com" -comm -h sha1 -in "${win_file}" \
                 -out "${win_file}.firstpass"
    rm -f "${win_file}"
    echo "==> Signing Windows binary (second pass)"
    osslsigncode sign -pkcs12 "${win_sig_file}" -pass "${SignKeyPassword}" -n "Vagrant VMware Utility" \
                 -i "https://www.vagrantup.com" -t "http://timestamp.digicert.com" -comm -nest -h sha256 \
                 -in "${win_file}.firstpass" -out "${win_file}"
else
    echo "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!"
    echo "! Windows binary is unsigned !"
    echo "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!"
fi

echo "==> Relocating built binaries..."

mkdir -p "${binary_storage}"
mv -f "${vmware_utility}/vagrant-vmware-utility_"* "${binary_storage}/" ||
    fail "Failed to move windows utility binary to binary storage"

echo "==> Vagrant VMware Utility binaries located in: ${binary_storage}"
