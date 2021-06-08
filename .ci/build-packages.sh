#!/usr/bin/env bash

export GO_VERSION="1.15.8"
export SLACK_USERNAME="Vagrant VMware Utility"
export SLACK_ICON="https://avatars.slack-edge.com/2017-10-17/257000837696_070f98107cdacc0486f6_36.png"
export SLACK_TITLE="📦 System Packaging"
export SLACK_CHANNEL="#team-vagrant"
export PACKET_EXEC_DEVICE_NAME="${PACKET_EXEC_DEVICE_NAME:-ci-utility-installers}"
export PACKET_EXEC_DEVICE_SIZE="${PACKET_EXEC_DEVICE_SIZE:-baremetal_0,baremetal_1,baremetal_1e}"
export PACKET_EXEC_PREFER_FACILITIES="${PACKET_EXEC_PREFER_FACILITIES:-iad1,iad2,ewr1,dfw1,dfw2,sea1,sjc1,lax1}"
export PACKET_EXEC_OPERATING_SYSTEM="${PACKET_EXEC_OPERATING_SYSTEM:-ubuntu_18_04}"
export PACKET_EXEC_PRE_BUILTINS="${PACKET_EXEC_PRE_BUILTINS:-InstallVmware,InstallVagrant,InstallVagrantVmware}"
export PACKET_EXEC_ATTACH_VOLUME="1"
export PACKET_EXEC_QUIET="1"
export PACKET_EXEC_PERSIST="1"
export PKT_VAGRANT_HOME="/mnt/data"
export PKT_VAGRANT_CLOUD_TOKEN="${VAGRANT_CLOUD_TOKEN}"
export PKT_DEBIAN_FRONTEND="noninteractive"

container_name="vagrant-vmware-utility-worker"
csource="${BASH_SOURCE[0]}"
while [ -h "$csource" ] ; do csource="$(readlink "$csource")"; done
root="$( cd -P "$( dirname "$csource" )/../" && pwd )"

. "${root}/.ci/load-ci.sh"

# Since we have a custom tagging strategy in this repository,
# clean up the tag and re-check if this is a release event.
if [ ! -z "${tag}" ]; then
    release_version="${tag#utility-v}"
    valid_release_version "${release_version}"
    if [ $? -eq 0 ]; then
        release=1
    fi
fi

# Job ID gets provided via common.sh
export PACKET_EXEC_REMOTE_DIRECTORY="${job_id}"

function cleanup {
    # Run these directly since we don't care about failure, simply
    # a best attempt.
    packet-exec run -quiet -- vagrant destroy --force > "${output}" 2>&1
    unset PACKET_EXEC_PERSIST
    packet-exec run -quiet -- docker kill "${container_name}" > "${output}" 2>&1
}

# Ensure we are in the root directory of the repository
pushd "${root}" > "${output}"

# Notify that the build is getting started
if [ "${release}" = "1" ]; then
    slack -m "📢 New vagrant-vmware-utility release has been triggered"
else
    slack -m "New vagrant-vmware-utility package build has been triggered"
fi

echo "++> Creating packet device if needed..."
packet-exec info

if [ $? -ne 0 ]; then
    wrap_stream packet-exec create \
                "Failed to create packet device"
fi

# Make signing files available before upload
secrets=$(load-signing) || fail "Failed to load signing files"
eval "${secrets}"

echo "++> Setting up packet device..."
wrap_stream packet-exec run -upload -- apt-get update -q \
            "Failed to setup packet device (package metadata update)"
pkt_wrap_stream apt-get install -yq osslsigncode docker.io \
                "Failed to install required system packages"

# Only need the builtins to run on the first execution
unset PACKET_EXEC_PRE_BUILTINS

# Install golang if not already installed
packet-exec run -quiet -- test -e /usr/local/bin/go
if [ $? -ne 0 ]; then
    pkt_wrap_stream curl -o /tmp/go.tgz -Ls \
                    "https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz" \
                    "Failed to setup packet device (golang download)"
    pkt_wrap_stream tar -C /usr/local -xzf /tmp/go.tgz \
                    "Failed to setup packet device (golang unpack)"
    pkt_wrap ln -s "/usr/local/go/bin/*" /usr/local/bin/ \
             "Failed to setup packet device (go links)"
fi

pkt_wrap go get github.com/mitchellh/gox \
         "Failed to install gox"
pkt_wrap 'ln -sf /root/go/bin/gox /usr/local/bin/gox' \
         "Failed to symlink local go binaries"

echo "++> Build binaries..."

# Load our secrets
export PACKET_EXEC_PRE_BUILTINS="LoadSecrets"

wrap_stream packet-exec run -download "pkg/binaries/*:pkg/binaries/" -- \
            ./package/init.sh \
            "Failed to build utility binaries"

unset PACKET_EXEC_PRE_BUILTINS

wrap chmod 0755 pkg/binaries/* \
     "Failed to update permissions on utility binaries"

binary=(pkg/binaries/*linux_amd64)
binary="$(printf "%s" "${binary}")"
utility_version="$("${binary}" -v 2>&1)"

if [ $? -ne 0 ]; then
    fail "Unable to read utility version from binary: ${utility_version}"
fi

echo "Local binary: ${binary} @ version: ${utility_version}"

echo "++> Starting DEB and RPM build container..."
pkt_wrap_stream docker run -v "\$(pwd):/vagrant" --name "${container_name}" \
                --workdir /vagrant --rm -i -d ubuntu:18.04 bash \
                "Failed to build DEB/RPM container"

echo "++> Installing required packages..."
pkt_wrap_stream docker exec "${container_name}" apt-get update \
                "Container setup failed - DEB/RPM"
pkt_wrap_stream docker exec "${container_name}" apt-get install -yq \
                ruby ruby-dev build-essential rpm autoconf libtool git \
                "Container setup failed - DEB/RPM"
pkt_wrap_stream docker exec "${container_name}" gem install \
                --no-document fpm \
                "Container setup failed - DEB/RPM"

echo "++> Starting DEB and RPM builds..."
wrap_stream packet-exec run -download "pkg/binaries/*:pkg/binaries" -- docker exec \
            "${container_name}" /vagrant/package/deb.sh \
            "Failed to build DEB package"
wrap_stream packet-exec run -download "pkg/binaries/*:pkg/binaries"  -- docker exec \
            "${container_name}" /vagrant/package/rpm.sh \
            "Failed to build RPM package"

echo "++> Destroying DEB and RPM build container..."
pkt_wrap_stream docker kill "${container_name}" \
                "Failed to kill container"

# name="${name}-arch"

# echo "++> Starting XZ build container..."
# docker run -v $(pwd):/vagrant --name $name --workdir /vagrant --rm -i -d archimg/base-devel bash

# # NOTE: don't need this currently with image used
# # echo "++> Installing required packages..."
# # docker exec ${name} pacman --noconfirm -Suy base-devel

# echo "++> Setting up container for build..."
# docker exec ${name} useradd dummy

# # Open up the perms so the dummy user can write
# chmod 777 ./pkg

# echo "++> Starting XZ build..."
# docker exec ${name} sudo -u dummy /vagrant/package/xz.sh

echo "++> Creating linux zip asset..."
tmpdir=$(mktemp --directory)
cp "${binary}" "${tmpdir}/vagrant-vmware-utility"
zip -j "pkg/vagrant-vmware-utility_${utility_version}_linux_amd64.zip" "${tmpdir}/"*
rm -rf "${tmpdir}"

echo "++> Upload zip package to remote..."
wrap_stream packet-exec run -upload -- /bin/true \
            "Failed to upload zip package to remote"

echo "++> Starting MSI and DMG builds..."
pkt_wrap_stream vagrant box update \
                "Failed to update Vagrant boxes"
pkt_wrap_stream vagrant box prune \
                "Failed to prune Vagrant boxes"

# Load our secrets
export PACKET_EXEC_PRE_BUILTINS="LoadSecrets"

pkt_wrap_stream vagrant up \
                "Failed to build packages"

# Clean out any persisted local assets
rm -rf ./pkg
mkdir -p ./pkg

wrap_stream packet-exec run -download "./pkg/*.*:./pkg" -- ./package/sign.sh "${utility_version}" \
            "Failed to sign utility installer packages"

unset PACKET_EXEC_PRE_BUILTINS

echo "++> Storing all assets..."
pushd ./pkg > "${output}"

upload_assets .


popd > "${output}"

echo "++> Generating release..."

export GITHUB_TOKEN="${HASHIBOT_TOKEN}"

if [ ! -z "${release}" ]; then
    release "${tag}" "./pkg"
    hashicorp_release "./pkg" "vagrant-vmware-utility"
    slack -m "New Vagrant release has been published! - *${utility_version}*\n\nAssets: https://releases.hashicorp.com/vagrant-vmware-utility/${utility_version}"
else
    prerelease_version=$(prerelease "utility-v${utility_version}" "./pkg")
    slack -m "New Vagrant VMware Utility development installers available:\n> https://github.com/${repository}/releases/${prerelease_version}"
fi

echo "++> Build complete!"

popd > "${output}"

exit $result
