require_relative "../spec_base.rb"
require "vagrant-vmware-desktop"

describe HashiCorp::VagrantVMwareDesktop do
  describe ".validate_install!" do
    let(:specifications){ [] }
    before do
      allow(Gem::Specification).to receive(:find_all_by_name).with(/vagrant-vmware/).
        and_return(specifications)
      allow(File).to receive(:exist?).with(/data.vmware-desktop-init/).
        and_return(false)
    end

    context "with no plugin conflicts" do
      it "should not raise error" do
        expect{ described_class.validate_install! }.not_to raise_error
      end
    end

    context "with plugin conflict" do
      let(:specifications){ ["result"] }

      it "should write error message to STDERR" do
        expect(STDERR).to receive(:puts).with(/incompatible plugin/)
        expect{ described_class.validate_install! }.to raise_error(StandardError)
      end

      it "should raise an error" do
        allow(STDERR).to receive(:puts)
        expect{ described_class.validate_install! }.to raise_error(StandardError)
      end
    end
  end

  describe ".orphan_artifact_cleanup!" do
    let(:init_exists){ true }

    before do
      allow(Vagrant::Util::Platform).to receive(:windows).and_return(false)
    end

    after{ described_class.orphan_artifact_cleanup! }

    let(:glob_results){ [] }

    before{ allow(Dir).to receive(:glob).and_return(glob_results) }

    context "with no directory matches" do
      it "should not attempt any directory removal" do
        expect(File).not_to receive(:rm_rf)
      end
    end

    context "with directory matches" do
      let(:glob_results){ ["RESULT1", "RESULT2"] }

      before do
        allow(File).to receive(:directory?).and_return(true)
        allow(FileUtils).to receive(:rm_rf)
      end

      it "should remove each result found" do
        glob_results.each{|res| expect(FileUtils).to receive(:rm_rf).with(res) }
      end

      it "should check that result is a directory before deletion" do
        glob_results.each{|res| expect(File).to receive(:directory?).with(res) }
      end
    end
  end
end
