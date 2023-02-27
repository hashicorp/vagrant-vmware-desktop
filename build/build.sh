#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

set -e

# Gem
echo "==> Building gem..."
gem build vagrant-vmware-desktop.gemspec
