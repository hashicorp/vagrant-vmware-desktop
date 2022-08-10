#!/usr/bin/env bash

function fail() {
    echo "ERROR: ${1}"
    exit 1
}

extension="dmg"

# Get our directory
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"

. "${DIR}/common.sh"

# Set information used for package and code signing
PKG_SIGN_IDENTITY=${VAGRANT_PACKAGE_SIGN_IDENTITY:-D38WU7D763}
PKG_SIGN_CERT_PATH=${VAGRANT_PACKAGE_SIGN_CERT_PATH:-"/Users/vagrant/MacOS_PackageSigning.cert"}
PKG_SIGN_KEY_PATH=${VAGRANT_PACKAGE_SIGN_KEY_PATH:-"/Users/vagrant/MacOS_PackageSigning.key"}

CODE_SIGN_IDENTITY=${VAGRANT_CODE_SIGN_IDENTITY:-D38WU7D763}
CODE_SIGN_CERT_PATH=${VAGRANT_CODE_SIGN_CERT_PATH:-"/Users/vagrant/MacOS_CodeSigning.p12"}
SIGN_KEYCHAIN=${VAGRANT_SIGN_KEYCHAIN:-/Library/Keychains/System.keychain}

echo "==> Installing package resources..."
pkg_contents="${base}/pkg-contents"
pkg_resources="${pkg_contents}/resources"
mkdir -p "${pkg_resources}"
cp "${package}/dmg/background.png" "${pkg_resources}/background.png" ||
    fail "Could not add installer background image"
cp "${package}/dmg/welcome.html" "${pkg_resources}/welcome.html" ||
    fail "Could not add installer welcome page"
cp "${package}/dmg/license.html" "${pkg_resources}/license.html" ||
    fail "Could not add installer license page"

echo "==> Creating postinstall script..."
mkdir -p "${pkg_contents}/scripts"
cat <<EOF >"${pkg_contents}/scripts/postinstall"
#!/usr/bin/env bash

/opt/vagrant-vmware-desktop/bin/vagrant-vmware-utility service uninstall
/opt/vagrant-vmware-desktop/bin/vagrant-vmware-utility certificate generate
/opt/vagrant-vmware-desktop/bin/vagrant-vmware-utility service install

chflags hidden /opt
exit 0
EOF
chmod 0755 "${pkg_contents}/scripts/postinstall" ||
    fail "Could not modify permission of post install script"

# Install and enable package signing if available
if [[ -f "${PKG_SIGN_CERT_PATH}" && -f "${PKG_SIGN_KEY_PATH}" ]]
then
    echo "==> Installing package signing key..."
    security import "${PKG_SIGN_CERT_PATH}" -k "${SIGN_KEYCHAIN}" -T /usr/bin/codesign -T /usr/bin/pkgbuild -T /usr/bin/productbuild ||
        fail "Failed to import package signing certificate"
    security import "${PKG_SIGN_KEY_PATH}" -k "${SIGN_KEYCHAIN}" -T /usr/bin/codesign -T /usr/bin/pkgbuild -T /usr/bin/productbuild ||
        fail "Failed to import package signing key"
    SIGN_PKG="1"
fi

# Install and enable code signing if available
if [[ -f "${CODE_SIGN_CERT_PATH}" && "${CODE_SIGN_PASS}" != "" ]]
then
    echo "==> Installing code signing key..."
    security import "${CODE_SIGN_CERT_PATH}" -k "${SIGN_KEYCHAIN}" -P "${CODE_SIGN_PASS}" -T /usr/bin/codesign ||
        fail "Failed to import code signing certificate"
    SIGN_CODE="1"
fi
set -e

if [[ "${SIGN_CODE}" -eq "1" ]]
then
    echo "==> Signing executables..."
    find "${stage}" -type f -perm +0111 -exec codesign --options=runtime -s "${CODE_SIGN_IDENTITY}" {} \; ||
        fail "Failed to sign all executables"
fi

if [[ "${SIGN_PKG}" -eq "1" ]]
then
    echo "==> Building signed core.pkg..."
    pkgbuild \
        --root "${stage}/opt/vagrant-vmware-desktop" \
        --identifier com.vagrant.vagrant-vmware-utility \
        --version "${version}" \
        --install-location "/opt/vagrant-vmware-desktop" \
        --scripts "${pkg_contents}/scripts" \
        --timestamp=none \
        --sign "${PKG_SIGN_IDENTITY}" \
        "${pkg_contents}/core.pkg" ||
        fail "Failed to build core package"
else
    echo "==> Building unsigned core.pkg..."
    pkgbuild \
        --root "${stage}/opt/vagrant-vmware-desktop" \
        --identifier com.vagrant.vagrant-vmware-utility \
        --version "${version}" \
        --install-location "/opt/vagrant-vmware-desktop" \
        --scripts "${pkg_contents}/scripts" \
        --timestamp=none \
        "${pkg_contents}/core.pkg" ||
        fail "Failed to build core package"
fi

cat <<EOF >"${pkg_contents}/vagrant-vmware-utility.dist"
<installer-gui-script minSpecVersion="1">
    <title>Vagrant VMware Utility</title>

    <!-- Configure the visuals and the various pages that exist throughout
         the installation process. -->
    <background file="background.png"
        alignment="bottomleft"
        mime-type="image/png" />
    <welcome file="welcome.html"
        mime-type="text/html" />
    <license file="license.html"
        mime-type="text/html" />

    <!-- Don't let the user customize the install (i.e. choose what
         components to install. -->
    <options customize="never" />

    <!-- The "choices" for things that can be installed, although the
         user has no actually choice, they're still required so that
         the installer knows what to install. -->
    <choice description="Vagrant VMware Utility"
        id="choice-vagrant-vmware-utility"
        title="Vagrant VMware Utility">
        <pkg-ref id="com.vagrant.vagrant-vmware-utility">core.pkg</pkg-ref>
    </choice>

    <choices-outline>
        <line choice="choice-vagrant-vmware-utility" />
    </choices-outline>
</installer-gui-script>
EOF

mkdir -p "${pkg_contents}/dmg"

# Check is signing certificate is available. Install
# and sign if found.
if [[ "${SIGN_PKG}" -eq "1" ]]
then
    echo "==> Building signed VagrantVMwareUtility.pkg..."

    productbuild \
        --distribution "${pkg_contents}/vagrant-vmware-utility.dist" \
        --resources "${pkg_resources}" \
        --package-path "${pkg_contents}" \
        --timestamp=none \
        --sign "${PKG_SIGN_IDENTITY}" \
        "${pkg_contents}/dmg/VagrantVMwareUtility.pkg" ||
        fail "Failed to build final package"
else
    echo "==> Building unsigned VagrantVMwareUtility.pkg..."

    productbuild \
        --distribution "${pkg_contents}/vagrant-vmware-utility.dist" \
        --resources "${pkg_resources}" \
        --package-path "${pkg_contents}" \
        --timestamp=none \
        "${pkg_contents}/dmg/VagrantVMwareUtility.pkg" ||
        fail "Failed to build final package"
fi

echo "==> Installing uninstall.tool..."

cp "${package}/dmg/uninstall.tool" "${pkg_contents}/dmg/uninstall.tool" ||
    fail "Could not add uninstall script"
chmod +x "${pkg_contents}/dmg/uninstall.tool" ||
    fail "Could not modify permissions on uninstall script"

echo "==> Creating DMG..."

tmp_output="/tmp/$(basename "${asset}")"

dmgbuild -s "${package}/dmg/dmgbuild.py" \
    -D srcfolder="${pkg_contents}/dmg" \
    -D backgroundimg="${package}/dmg/background_installer.png" \
    "Vagrant VMware Utility" \
    "${tmp_output}" ||
    fail "Failed to build final DMG"

if [[ "${SIGN_PKG}" -ne "1" ]]
then
    echo
    echo "!!!!!!!!!!!! WARNING !!!!!!!!!!!!"
    echo "! Vagrant installer package is  !"
    echo "! NOT signed. Rebuild using the !"
    echo "! signing key for release build !"
    echo "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!"
    echo
else
    echo "==> Signing DMG..."
    codesign -s "${PKG_SIGN_IDENTITY}" --timestamp "${tmp_output}" ||
        fail "Failed to sign final DMG"
fi

if [ "${SIGN_PKG}" = "1" ] && [ "${SIGN_CODE}" = "1" ] && [ "${NOTARIZE_USERNAME}" != "" ]; then
    echo "==> Notarizing DMG..."
    export AC_USERNAME="${NOTARIZE_USERNAME}"
    export AC_PASSWORD="${NOTARIZE_PASSWORD}"
    cat <<EOF > config.hcl
notarize {
  path = "${tmp_output}"
  bundle_id = "com.hashicorp.vagrant.vagrant-vmware-utility"
  staple = true
}
EOF
    gon ./config.hcl || fail "Failed to notarize final DMG"
else
    echo
    echo "!!!!!!!!!!!!WARNING!!!!!!!!!!!!!!!!!!"
    echo "! Vagrant installer package is NOT  !"
    echo "! notarized. Rebuild with proper    !"
    echo "! signing and credentials to enable !"
    echo "! package notarization.             !"
    echo "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!"
fi

cp -f "${tmp_output}" "${asset}" ||
    fail "Could not copy DMG to final destination"

echo "==> Cleaning up package artifacts..."
rm -rf "${pkg_contents}" "${pkg_resources}" "${stage}" "${tmp_output}"

echo "==> Package build complete: ${asset}"
