#!/bin/bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0


function fail() {
    echo "ERROR: ${1}"
    exit 1
}

# Get the parent directory of where this script is.
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )/.." && pwd )"

# Change into that dir because we expect that
pushd "${DIR}" > /dev/null 2>&1 || fail "Could not enter project directory"

# Get the version from the command line
VERSION="${1}"
if [ -z "${VERSION}" ]; then
    echo "Please specify a version."
    exit 1
fi

if [[ ! "${VERSION}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Invalid version format provided. Expecting X.X.X format."
    exit 1
fi

if [ ! -d "./pkg" ]; then
    echo "Package directory (.pkg) does not exist."
    exit 1
fi

# If the binaries directory is found, just delete it
if [ -d "./pkg/binaries" ]; then
    echo "Removing build artifact directory ./pkg/binaries"
    rm -rf ./pkg/binaries
fi

# Same with init
if [ -d "./pkg/init" ]; then
    echo "Removing init artifact directory ./pkg/init"
    rm -rf ./pkg/init
fi
