# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

require_relative "../../spec_base"
require "vagrant"

require "vagrant-vmware-desktop/cap/snapshot"

describe HashiCorp::VagrantVMwareDesktop::Cap::Snapshot do
  let(:subject) { described_class }

  let(:driver) { double("driver") }

  let(:machine) { double("machine", provider: double("provider", driver: driver))}

  describe "#delete_all_snapshots" do

    it "deletes the snapshots" do
      snapshot_tree = ["a", "b", "a/child", "a/child/superchild"]
      allow(driver).to receive(:snapshot_tree).and_return(snapshot_tree)
      expect(driver).to receive(:snapshot_delete).exactly(4)
      subject.delete_all_snapshots(machine)
    end

    it "does not delete snapshots if none exist" do
      snapshot_tree = []
      allow(driver).to receive(:snapshot_tree).and_return(snapshot_tree)
      expect(driver).to_not receive(:snapshot_delete)
      subject.delete_all_snapshots(machine)
    end
  end

end
