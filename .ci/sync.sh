#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0


csource="${BASH_SOURCE[0]}"
while [ -h "$csource" ] ; do csource="$(readlink "$csource")"; done
root="$( cd -P "$( dirname "$csource" )/../" && pwd )"

. "${root}/.ci/load-ci.sh"

export PATH="${PATH}:${root}/.ci"

pushd "${root}"

# Configure for hashibot
hashibot_git

if [ "${repo_name}" = "vagrant-vmware-desktop" ]; then
    remote_repository="hashicorp/vagrant-vmware-desktop-builder"
else
    remote_repository="hashicorp/vagrant-vmware-desktop"
fi

wrap git config pull.rebase false \
     "Failed to configure git pull strategy"


echo "Adding remote mirror repository '${remote_repository}'..."
wrap git remote add mirror "https://${HASHIBOT_USERNAME}:${HASHIBOT_TOKEN}@github.com/${remote_repository}" \
     "Failed to add mirror '${remote_repository}' for sync"

echo "Updating configured remotes..."
wrap_stream git remote update mirror \
            "Failed to update mirror repository (${remote_repository}) for sync"

rb=$(git branch -r --list "mirror/${ident_ref}")

if [ "${rb}" != "" ]; then
    echo "Pulling ${ident_ref} from mirror..."
    wrap_stream git pull mirror "${ident_ref}" \
                "Failed to pull ${ident_ref} from mirror repository (${remote_repository}) for sync"
fi

echo "Pushing ${ident_ref} to mirror..."
wrap_stream git push mirror "${ident_ref}" \
            "Failed to sync mirror repository (${remote_repository})"

# Push tags _only_ to the builder repository
if [ "${repo_name}" = "vagrant-vmware-desktop-builder" ] || [ -z "${tag}" ]; then
    exit
fi

echo "Pushing tag ${tag} to mirror..."
wrap_stream git push mirror "${tag}" \
    "Failed to push tag ${tag} to mirror repository (${remote_repository})"
