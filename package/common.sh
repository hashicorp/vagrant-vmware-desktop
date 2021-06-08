# Get our directory
csource="${BASH_SOURCE[0]}"
while [ -h "$csource" ] ; do csource="$(readlink "$csource")"; done
package="$( cd -P "$( dirname "$csource" )" && pwd )"

if [[ "${extension}" = "" ]]; then
    echo "ERROR: extension variable is unset!"
    exit 1
fi

root=$(dirname "${package}")
base="${root}/pkg"
stage="${base}/${extension}"
binary_storage="${base}/binaries"

if [ "${extension}" = "dmg" ]; then
    binary_path="${binary_storage}/vagrant-vmware-utility_darwin_amd64"
else
    binary_path="${binary_storage}/vagrant-vmware-utility_linux_amd64"
fi

if [ ! -f "${binary_path}" ]; then
    echo "!!! No VMware utility binary found. Please run ${package}/init.sh!"
    exit 1
fi

echo "==> Setting up for ${extension} build..."

if [ -d "${stage}" ]; then
    echo "    !! Removing existing ${stage} directory"
    rm -rf "${stage}"
fi

mkdir -p "${stage}/opt/vagrant-vmware-desktop/bin"

echo "==> Installing vagrant-vmware-utility..."

cp -f "${binary_path}" "${stage}/opt/vagrant-vmware-desktop/bin/vagrant-vmware-utility"
chmod -R 755 "${stage}/opt/vagrant-vmware-desktop"

if [ "${UTILITY_VERSION}" != "" ]; then
    version="${UTILITY_VERSION}"
else
    echo -n "==> Detecting utility version... "
    version=$("${stage}/opt/vagrant-vmware-desktop/bin/vagrant-vmware-utility" -v 2>&1)
    if [ $? -ne 0 ]; then
        echo "!!! Failed to read utility version"
        exit 1
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
        -init-style "sysv" > "${base}/init/vagrant-vmware-utility.init"

    # Generate systemd file
    "${stage}/opt/vagrant-vmware-desktop/bin/vagrant-vmware-utility" \
        service install \
        -print \
        -config-write "${stage}/opt/vagrant-vmware-desktop/config/service.hcl" \
        -config-path "/opt/vagrant-vmware-desktop/config/service.hcl" \
        -exe-path "/opt/vagrant-vmware-desktop/bin/vagrant-vmware-utility" \
        -init-style "systemd" > "${base}/init/vagrant-vmware-utility.service"
fi

asset="${base}/vagrant-vmware-utility_${version}_x86_64.${extension}"
