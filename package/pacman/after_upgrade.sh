#!/usr/bin/env bash

/opt/vagrant-vmware-desktop/bin/vagrant-vmware-utility certificate generate || exit
/opt/vagrant-vmware-desktop/bin/vagrant-vmware-utility service install
