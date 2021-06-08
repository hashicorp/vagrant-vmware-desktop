#!/usr/bin/env bash
set -e

# Gem
echo "==> Building gem..."
gem build vagrant-vmware-desktop.gemspec
