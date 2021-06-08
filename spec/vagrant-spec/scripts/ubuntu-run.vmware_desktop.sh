#!/bin/bash
set -x

export VAGRANT_SPEC_BOX="${VAGRANT_SPEC_BOX}"
sudo --login -u vagrant VAGRANT_SPEC_BOX="${VAGRANT_SPEC_BOX}" VAGRANT_LICENSE_FILE="/tmp/vmware.lic" \
     VAGRANT_VMWARE_PLUGIN_FILE="/vagrant/vagrant-vmware.gem" vagrant vagrant-spec ${VAGRANT_SPEC_ARGS} \
     /vagrant/spec/vagrant-spec/configs/vagrant-spec.config.vmware.rb
result=$?

exit $result
