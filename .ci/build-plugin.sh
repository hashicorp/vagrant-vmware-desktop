#!/usr/bin/env bash

export SLACK_USERNAME="Vagrant VMware Desktop"
export SLACK_ICON="https://avatars.slack-edge.com/2017-10-17/257000837696_070f98107cdacc0486f6_36.png"
export SLACK_TITLE="💎 RubyGem Building"
export SLACK_CHANNEL="#team-vagrant"

csource="${BASH_SOURCE[0]}"
while [ -h "$csource" ] ; do csource="$(readlink "$csource")"; done
root="$( cd -P "$( dirname "$csource" )/../" && pwd )"

. "${root}/.ci/load-ci.sh"

slack -m "📢 New vagrant-vmware-desktop RubyGem build has been triggered"

wrap_raw pushd "${root}"

# Read the version we are building
version="$(<./versions/desktop.txt)"

# Build the gem
wrap_stream ./build/build.sh \
            "Failed to build vagrant-vmware-desktop development RubyGem v${version}"

# Get the path of our new gem
g=(vagrant*.gem)
gem=$(printf "%s" "${g}")

# Upload built gem to the asset store
set -x
upload_assets "${gem}"
set +x
slack -m "New development version of vagrant-vmware-desktop available: v${version}\n* $(asset_location)/${gem}"
