#!/usr/bin/env bash

set -e

export PATH="/usr/local/bin:$PATH"

sudo pip3 install dmgbuild
curl -Lo gon.zip https://github.com/mitchellh/gon/releases/download/v0.2.2/gon_0.2.2_macos.zip
unzip gon.zip
chown root:wheel gon
chmod 755 gon
mv gon /System/Volumes/Data/usr/local/bin/gon

/vagrant/package/dmg.sh
