## Releasing

This documents how to release the Vagrant VMware desktop plugin and the
Vagrant VMware utility. Some steps in this document require privileged
access to private systems. This document is targeted to Vagrant core
members with the ability to start the release process.

This repository contains two separate projects:

* Vagrant VMware Desktop
* Vagrant VMware Utility

The release process for these projects are similiar but slightly different based on the
project.

### Plugin (vagrant-vmware-desktop)

The steps for releasing the plugin are as follows:

1. Ensure the repository is up-to-date and the main branch is checked out
1. Update the version for release (located in: ./versions/desktop.txt)
1. Update the `CHANGELOG-plugin.md` with the current date
1. Commit the version file and CHANGELOG update
1. Create a new tag for the release in the format: desktop-vX.X.X
1. Push the changes: git push origin main desktop-vX.X.X

Once the new tag has been pushed, a release task will be triggered in the Vagrant CI system. After the release has been completed, add the latest version to checkpoint.

### Utility (vagrant-vmware-utility)

The steps for releasing the utility are as follows:

1. Ensure the repository is up-to-date and the main branch is checked out
1. Update the version for release (located in: ./go_src/vagrant-vmware-utility/version/version.go)
1. Update the `CHANGELOG-utility.md` with the current date
1. Commit the version file and CHANGELOG update
1. Create a new tag for the release in the format: utility-vX.X.X
1. push the changes: git push origin main utility-vX.X.X

Once the new tag has been pushed, a release task will be triggered in the Vagrant CI system. After the release has been completed:

1. Change to the `hashicorp/vagrant` repository
1. Open the ./website/data/version.json file
1. Update the version for VMWARE_UTILITY_VERSION
1. Commit the change to main and push
1. Checkout the stable-website branch and cherrypick the version update commit
1. Push the stable-website branch to update the published version
1. Finally, update checkpoint with the new utility version
