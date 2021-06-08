#!/usr/bin/env bash

export SLACK_USERNAME="Vagrant VMware Desktop"
export SLACK_ICON="https://avatars.slack-edge.com/2017-10-17/257000837696_070f98107cdacc0486f6_36.png"
export SLACK_TITLE="💎 RubyGems Publishing"
export SLACK_CHANNEL="#team-vagrant"

csource="${BASH_SOURCE[0]}"
while [ -h "$csource" ] ; do csource="$(readlink "$csource")"; done
root="$( cd -P "$( dirname "$csource" )/../" && pwd )"

. "${root}/.ci/load-ci.sh"

slack -m "📢 New vagrant-vmware-desktop release has been triggered"

wrap_raw pushd "${root}"

# Read the version we are building
version="$(<./versions/desktop.txt)"

# Check if we are a valid version to release
valid_release_version "${version}"
if [ $? -ne 0 ]; then
    fail "Invalid version format for vagrant-vmware-desktop release: ${version}"
fi

# Build and publish our gem
publish_to_rubygems

slack -m "New version of vagrant-vmware-desktop published: v${version}"
