#!/bin/bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

set -xe

apt-get update -yq
apt-get install -yq build-essential libxinerama1 libxcursor1 libxtst6 libxi6 linux-kernel-headers open-vm-tools
apt-get install -yq linux-headers-$(uname -a | awk '{print $3}')

pushd /vagrant

if [ ! -f /vagrant/vmware-installer ]
then
    curl -L -o /vagrant/vmware-installer https://www.vmware.com/go/tryworkstation-linux-64
fi
chmod 755 /vagrant/vmware-installer

# do installer magic to make things compile
sed 's/gcc version 6/gcc version 5/' /proc/version > /tmp/gcc-version-proc
mount --bind /tmp/gcc-version-proc /proc/version
/vagrant/vmware-installer --eulas-agreed --required
vmware-modconfig --console --install-all
umount /proc/version
rm /tmp/gcc-version-proc

/usr/lib/vmware/bin/vmware-vmx-debug --new-sn ${VMWARE_SN}

dpkg -i ./pkg/dist/vagrant_*_x86_64.deb

if [ ! -f /tmp/vmware.lic ]
then
    curl -L -o /tmp/vmware.lic "${VAGRANT_VMWARE_LICENSE_URL}"
fi

sudo --login -u vagrant vagrant plugin install /vagrant/vagrant-spec.gem
sudo --login -u vagrant vagrant plugin install /vagrant/vagrant-vmware.gem

sudo --login -u vagrant vagrant plugin license vagrant-vmware-workstation /tmp/vmware.lic

popd
