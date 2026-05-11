#!/usr/bin/env bash
# Copyright IBM Corp. 2021, 2026
# SPDX-License-Identifier: MPL-2.0

set -e

# Gem
echo "==> Building gem..."
gem build vagrant-vmware-desktop.gemspec
