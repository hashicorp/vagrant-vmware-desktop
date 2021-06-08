#!/bin/bash

# Get the parent directory of where this script is.
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )/.." && pwd )"

# Change into that dir because we expect that
cd "${DIR}"

# Get the version from the command line
VERSION=$1
if [ -z $VERSION ]; then
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

# Make the checksums
pushd ./pkg

shasum -a256 * > ./vagrant-vmware-utility_${VERSION}_SHA256SUMS

if [ $? -ne 0 ]; then
    echo "Failed to generate checksum values"
    exit 1
fi

if [ -z "${NOSIGN}" ]; then
    echo "==> Signing..."
    gpg --default-key 348FFC4C --detach-sig ./vagrant-vmware-utility_${VERSION}_SHA256SUMS

    if [ $? -ne 0 ]; then
        echo "Failed to create checksums signature file"
        exit 1
    fi
fi
popd
