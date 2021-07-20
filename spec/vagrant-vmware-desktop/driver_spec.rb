require "pathname"
require "vagrant-vmware-desktop"

describe HashiCorp::VagrantVMwareDesktop::Driver::Base do
  before do
    @vmx_file = Tempfile.new('vagrant-vmware-test')
    @vmx_file.print(vmx_contents)
    @vmx_file.close
  end
  after{ FileUtils.rm(@vmx_file) }

  let(:vmx_file){ @vmx_file }
  let(:vmx_contents){ "" }
  let(:provider_config){ double("provider_config", utility_host: "127.0.0.1",
    utility_port: 9922, utility_certificate_path: '/dev/null', nat_device: "vmnet8",
    force_vmware_license: nil) }
  let(:instance) { described_class.new(vmx_file.path, provider_config) }
  let(:utility_response){ HashiCorp::VagrantVMwareDesktop::Helper::VagrantUtility::Response }
  let(:utility_version) { "1.0" }
  let(:vagrant_utility){ double("vagrant_utility") }

  # Stub initial API requests
  before do
    allow_any_instance_of(described_class).to receive(:vagrant_utility).and_return(vagrant_utility)
    allow(HashiCorp::VagrantVMwareDesktop::Helper::VagrantUtility).to receive(:new).and_return(vagrant_utility)
    allow(vagrant_utility).to receive(:get).with("/vmware/info").and_return(utility_response.new(success: true))
    allow(vagrant_utility).to receive(:get).with("/version").and_return(utility_response.new(content: {version: utility_version}))
    allow(vagrant_utility).to receive(:get).with("/vmware/paths").and_return(
      utility_response.new(
        success: true,
        content: {
          vmrun: "VMRUN_PATH",
          vmx: "VMX_PATH"
        }
      )
    )
    allow_any_instance_of(HashiCorp::VagrantVMwareDesktop::Errors::Base).to receive(:translate_error)
  end

  describe "#detect_nat_device!" do
    let(:vmnet_devices) { [] }

    before do
      allow(provider_config).to receive(:nat_device=)
      allow(instance).to receive(:read_vmnet_devices).
        and_return(vmnet_devices)
    end

    context "with no vmnet devices are returned" do
      it "should return the default nat device" do
        expect(provider_config).to receive(:nat_device=).
          with(described_class.const_get(:DEFAULT_NAT_DEVICE))
        instance.detect_nat_device!
      end
    end

    context "with vmnet devices" do
      let(:vmnet_devices) { [
        {name: "vmnet1", type: "public"},
        {name: "vmnet2", type: "public"}
      ] }

      context "with no NAT devices" do
        it "should return the default nat device" do
          expect(provider_config).to receive(:nat_device=).
            with(described_class.const_get(:DEFAULT_NAT_DEVICE))
          instance.detect_nat_device!
        end
      end

      context "with non-default NAT device" do
        before { vmnet_devices << {name: "vmnet3", type: "nat", dhcp: true, hostonly_subnet: true} }

        it "should return the non-default NAT" do
          expect(provider_config).to receive(:nat_device=).
            with("vmnet3")
          instance.detect_nat_device!
        end

        context "with default NAT device" do
          before  do
            vmnet_devices << {
              name: described_class.const_get(:DEFAULT_NAT_DEVICE),
              type: "nat", dhcp: true, hostonly_subnet: true }
          end

          it "should return the default NAT" do
            expect(provider_config).to receive(:nat_device=).
              with(described_class.const_get(:DEFAULT_NAT_DEVICE))
            instance.detect_nat_device!
          end
        end
      end
    end
  end

  describe "#read_ip" do
    let(:guest_ip){ "192.168.99.10" }
    let(:vmrun_result){ double(stdout: guest_ip) }
    context "with vmrun ip lookup enabled" do
      before do
        expect(instance).to receive(:vmrun).with("getGuestIPAddress", vmx_file.path).and_return(vmrun_result)
      end

      it "should return guest IP via vmrun command" do
        expect(instance.read_ip).to eql(guest_ip)
      end

      it "should not perform DHCP lookup if vmrun returns IP" do
        expect(instance).to_not receive(:read_dhcp_leases)
        expect(instance.read_ip).to eql(guest_ip)
      end
    end

    context "with vmrun ip lookup disabled" do
      let(:mac){ 'MACADDR' }
      let(:vmx_contents) do
        "ethernet1.present = \"TRUE\"\n" \
          "ethernet1.connectiontype = \"nat\"\n" \
          "ethernet1.address = \"#{mac}\"\n"
      end
      before do
        expect(instance).to receive(:read_dhcp_lease).with("vmnet8", mac).and_return(guest_ip)
      end

      it "should return guest IP address via DHCP lookup" do
        expect(instance.read_ip(false)).to eql(guest_ip)
      end

      it "should not execute vmrun to lookup guest IP address" do
        expect(instance).to_not receive(:vmrun)
        expect(instance.read_ip(false)).to eql(guest_ip)
      end
    end

    context "with vmrun returning IP address ending with '.1'" do
      let(:guest_ip){ nil }
      let(:vmrun_guest_ip){ "10.0.0.1" }
      let(:vmrun_result){ double(stdout: vmrun_guest_ip) }
      let(:mac){ 'MACADDR' }
      let(:vmx_contents) do
        "ethernet1.present = \"TRUE\"\n" \
          "ethernet1.connectiontype = \"nat\"\n" \
          "ethernet1.address = \"#{mac}\"\n"
      end

      before do
        expect(instance).to receive(:read_dhcp_lease).with("vmnet8", mac).and_return(guest_ip)
        expect(instance).to receive(:vmrun).with("getGuestIPAddress", vmx_file.path).and_return(vmrun_result)
      end

      it "should discard vmrun IP result and perform DHCP lookup" do
        expect(instance.read_ip).to be_nil
      end
    end
  end

  describe "#display_ethernet_allowlist_warning" do
    before do
      allow($stderr).to receive(:puts)
      allow(File).to receive(:exist?).and_return(false)
      allow(FileUtils).to receive(:touch)
    end

    it "should output warning to STDERR" do
      expect($stderr).to receive(:puts).with(/CUSTOM-KEY/)
      instance.send(:display_ethernet_allowlist_warning, "CUSTOM-KEY", "VAL")
    end

    it "should create file file when displaying notification" do
      expect(FileUtils).to receive(:touch).with(/CUSTOM-KEY/)
      instance.send(:display_ethernet_allowlist_warning, "CUSTOM-KEY", "VAL")
    end

    it "should not output warning when file exists" do
      expect(File).to receive(:exist?).with(/CUSTOM-KEY/).and_return(true)
      expect($stderr).not_to receive(:puts).with(/CUSTOM-KEY/)
      instance.send(:display_ethernet_allowlist_warning, "CUSTOM-KEY", "VAL")
    end

    it "should show unique warning for each key" do
      expect($stderr).to receive(:puts).with(/CUSTOM-KEY1/)
      expect($stderr).to receive(:puts).with(/CUSTOM-KEY2/)
      instance.send(:display_ethernet_allowlist_warning, "CUSTOM-KEY1", "VAL")
      instance.send(:display_ethernet_allowlist_warning, "CUSTOM-KEY2", "VAL")
    end

  end

  describe "#verify_vmnet!" do
    let(:response) {
      HashiCorp::VagrantVMwareDesktop::Helper::VagrantUtility::Response.new(result)
    }
    let(:result) { {code: code, success: success} }
    let(:code) { 204 }
    let(:success) { true }

    before do
      allow(instance.vagrant_utility).to receive(:post).with("/vmnet/verify").and_return(response)
    end

    it "should run successfully when response code is 204" do
      instance.verify_vmnet!
    end

    context "when request is unsuccessful" do
      let(:success) { false }

      context "when response code is 404" do
        let(:code) { 404 }

        it "should not raise an error" do
          expect{ instance.verify_vmnet! }.not_to raise_error
        end
      end

      context "when response code is non-200" do
        let(:code) { 400 }

        it "should raise an error" do
          expect{ instance.verify_vmnet! }.to raise_error(
            HashiCorp::VagrantVMwareDesktop::Errors::VMNetDevicesWontStart)
        end
      end
    end
  end

  describe "#forward_ports" do
    let(:definitions) {
      [{device: "vmnet8", protocol: "tcp", host_port: 9999, guest_port: 22, guest_ip: "127.0.0.1"},
        {device: "vmnet8", protocol: "tcp", host_port: 8888, guest_port: 33, guest_ip: "127.0.0.1"}]
    }
    let(:response) { utility_response.new(success: true) }

    before do
      allow(vagrant_utility).to receive(:put).and_return(response)
    end

    it "should make a utility request for each port forward defined" do
      expect(vagrant_utility).to receive(:put).with(/portforward/, any_args).twice.and_return(response)
      instance.forward_ports(definitions)
    end

    it "should raise custom error when port forward utility request fails" do
      expect(vagrant_utility).to receive(:put).with(/portforward/, any_args).and_return(utility_response.new(success: false))
      expect { instance.forward_ports(definitions) }.to raise_error(HashiCorp::VagrantVMwareDesktop::Errors::DriverAPIPortForwardError)
    end

    context "when utility version is greater than 1.0.6" do
      let(:utility_version) { "1.0.7" }

      it "should only send one request per device" do
        expect(vagrant_utility).to receive(:put).with(/portforward/, any_args).once.and_return(response)
        instance.forward_ports(definitions)
      end
    end
  end

  describe "#scrub_forwarded_ports" do
    let(:ports) { [double("portfwd1"), double("portfwd2")] }
    let(:response) {
      HashiCorp::VagrantVMwareDesktop::Helper::VagrantUtility::Response.new(result)
    }
    let(:result) { {code: code, success: success} }
    let(:code) { 204 }
    let(:success) { true }

    before do
      allow(instance).to receive(:all_forwarded_ports).with(true).and_return(ports)
      allow(instance.vagrant_utility).to receive(:delete).with("/vmnet/vmnet8/portforward", any_args).and_return(response)
    end

    it "should make a request for each port forward" do
      expect(instance.vagrant_utility).to receive(:delete).with("/vmnet/vmnet8/portforward", ports.first).and_return(response)
      expect(instance.vagrant_utility).to receive(:delete).with("/vmnet/vmnet8/portforward", ports.last).and_return(response)
      instance.scrub_forwarded_ports
    end

    context "when an error is returned" do
      let(:success) { false }
      let(:code) { 400 }

      it "should raise an error" do
        expect { instance.scrub_forwarded_ports }.to raise_error(HashiCorp::VagrantVMwareDesktop::Errors::DriverAPIPortForwardPruneError)
      end
    end

    context "when no ports are forwarded" do
      let(:ports) { [] }

      it "should not attempt to delete ports" do
        expect(instance.vagrant_utility).not_to receive(:delete)
        instance.scrub_forwarded_ports
      end
    end

    context "when utility version is greater than 1.0.7" do
      let(:utility_version) { "1.0.8" }

      it "should only send one request to delete all port forwards" do
        expect(instance.vagrant_utility).to receive(:delete).with("/vmnet/vmnet8/portforward", ports).and_return(response)
        instance.scrub_forwarded_ports
      end
    end
  end

  describe "#stop" do
    it "should receive a stop request for the VM" do
      expect(instance).to receive(:vmrun).with("stop", anything, "soft", any_args)
      instance.stop
    end

    it "should include a timeout for the soft stop" do
      expect(instance).to receive(:vmrun).with("stop", any_args, hash_including(timeout: 15))
      instance.stop
    end

    it "should attempt a hard stop when an error is encountered" do
      allow_any_instance_of(HashiCorp::VagrantVMwareDesktop::Errors::VMRunError).to receive(:translate_error)
      expect(instance).to receive(:vmrun).and_raise(HashiCorp::VagrantVMwareDesktop::Errors::VMRunError)
      expect(instance).to receive(:vmrun).with("stop", anything, "hard", any_args)
      instance.stop
    end

    it "should attempt a hard stop when soft stop results in a timeout" do
      expect(instance).to receive(:vmrun).and_raise(Vagrant::Util::Subprocess::TimeoutExceeded.new(-1))
      expect(instance).to receive(:vmrun).with("stop", anything, "hard", any_args)
      instance.stop
    end

    it "should attempt a hard stop when requested" do
      expect(instance).to receive(:vmrun).with("stop", anything, "hard", any_args)
      instance.stop("hard")
    end
  end

  describe "#verify!" do
    let(:response) { HashiCorp::VagrantVMwareDesktop::Helper::VagrantUtility::Response }

    before do
      allow(vagrant_utility).to receive(:post).
        and_return(response.new(success: true))
      stub_const("HashiCorp::VagrantVMwareDesktop::Driver::Base::VAGRANT_UTILITY_VERSION_REQUIREMENT", "> 1.1")
    end

    it "should raise an error when the utility version does not meet the requirement" do
      expect(vagrant_utility).to receive(:get).with("/version").and_return(content: {version: "1.0"})
      expect { instance.verify! }.to raise_error(HashiCorp::VagrantVMwareDesktop::Errors::UtilityUpgradeRequired)
    end

    it "should not raise an error when the utility version meets the requirement" do
      expect(vagrant_utility).to receive(:get).with("/version").and_return(content: {version: "2.0"})
      expect { instance.verify! }.not_to raise_error
    end
  end

  describe "#reserve_dhcp_address" do
    let(:mac) { "MAC" }
    let(:ip) { "IP" }
    let(:device) { "DEVICE" }

    before { allow(vagrant_utility).to receive(:put).
        and_return(utility_response.new(success: true)) }

    it "should return true on successful request" do
      expect(instance.reserve_dhcp_address(ip, mac)).to eq(true)
    end

    it "should include MAC address in request path" do
      expect(vagrant_utility).to receive(:put).with(/#{mac}/)
      instance.reserve_dhcp_address(ip, mac)
    end

    it "should include IP address in request path" do
      expect(vagrant_utility).to receive(:put).with(/#{ip}/)
      instance.reserve_dhcp_address(ip, mac)
    end

    it "should include device in request path" do
      expect(vagrant_utility).to receive(:put).with(/#{device}/)
      instance.reserve_dhcp_address(ip, mac, device)
    end

    context "when request fails" do
      before { allow(vagrant_utility).to receive(:put).
          and_return(utility_response.new(success: false, content: {message: "error"})) }

      it "should raise error" do
        expect { instance.reserve_dhcp_address(ip, mac) }.
          to raise_error(HashiCorp::VagrantVMwareDesktop::Errors::DriverAPIAddressReservationError)
      end
    end
  end

  describe "#setup_adapters" do
    let(:vmx) { double("vmx") }
    let(:adapters) { [] }

    before do
      allow(vmx).to receive(:[]=)
      allow(vmx).to receive(:keys).and_return([])
      allow(instance).to receive(:vmx_modify).and_yield(vmx)
    end

    it "should add no adapters when no adapters are defined" do
      expect(vmx).not_to receive(:[]=)
      instance.setup_adapters(adapters)
    end

    context "with NAT adapter defined" do
      let(:adapters) {
        [{type: :nat, slot: 0}]
      }

      it "should setup adapter in VMX data" do
        expect(vmx).to receive(:[]=).with("ethernet0.present", "TRUE")
        expect(vmx).to receive(:[]=).with("ethernet0.connectiontype", "nat")
        expect(vmx).to receive(:[]=).with("ethernet0.virtualdev", "e1000")
        instance.setup_adapters(adapters)
      end

      it "should set MAC address to be automatically generated" do
        expect(vmx).to receive(:[]=).with("ethernet0.addresstype", "generated")
        instance.setup_adapters(adapters)
      end

      context "with MAC address defined" do
        let(:adapters) {
          [{type: :nat, slot: 0, mac_address: mac_address}]
        }
        let(:mac_address) { "MAC" }

        it "should set the MAC address in the VMX data" do
          expect(vmx).to receive(:[]=).with("ethernet0.address", mac_address)
          instance.setup_adapters(adapters)
        end

        it "should set the address type to static" do
          expect(vmx).to receive(:[]=).with("ethernet0.addresstype", "static")
          instance.setup_adapters(adapters)
        end
      end

      context "with custom vnet defined" do
        let(:adapters) {
          [{type: :nat, slot: 0, vnet: vnet_device}]
        }
        let(:vnet_device) { "VNET_DEVICE" }

        it "should set the vnet in the VMX data" do
          expect(vmx).to receive(:[]=).with("ethernet0.vnet", vnet_device)
          instance.setup_adapters(adapters)
        end
      end
    end
  end

  describe "#stop" do
    before do
      allow(instance).to receive(:vmrun)
    end

    it "should stop the guest vm" do
      expect(instance).to receive(:vmrun).with("stop", any_args)
      instance.stop
    end

    context "when command fails" do
      before do
        expect(instance).to receive(:vmrun).with("stop", any_args).
          and_raise(HashiCorp::VagrantVMwareDesktop::Errors::VMRunError)
      end

      it "should attempt to hard stop the guest vm" do
        expect(instance).to receive(:vmrun).with("stop", anything, "hard", any_args)
        instance.stop
      end

      context "when hard stop fails and guest is running" do
        before do
          expect(instance).to receive(:read_state).and_return(:running)
          expect(instance).to receive(:vmrun).with("stop", anything, "hard", any_args).
            and_raise(HashiCorp::VagrantVMwareDesktop::Errors::VMRunError)
        end

        it "should raise the failure" do
          expect { instance.stop }.to raise_error(HashiCorp::VagrantVMwareDesktop::Errors::VMRunError)
        end
      end

      context "when hard stop fails and guest is not running" do
        before do
          expect(instance).to receive(:read_state).and_return(:not_running)
          expect(instance).to receive(:vmrun).with("stop", anything, "hard", any_args).
            and_raise(HashiCorp::VagrantVMwareDesktop::Errors::VMRunError)
        end

        it "should not raise a failure" do
          expect { instance.stop }.not_to raise_error
        end
      end
    end

    context "when command times out" do
      before do
        expect(instance).to receive(:vmrun).with("stop", any_args).
          and_raise(Vagrant::Util::Subprocess::TimeoutExceeded.new(0))
      end

      it "should attempt to hard stop the guest vm" do
        expect(instance).to receive(:vmrun).with("stop", anything, "hard", any_args)
        instance.stop
      end

      context "when hard stop fails and guest is running" do
        before do
          expect(instance).to receive(:read_state).and_return(:running)
          expect(instance).to receive(:vmrun).with("stop", anything, "hard", any_args).
            and_raise(HashiCorp::VagrantVMwareDesktop::Errors::VMRunError)
        end

        it "should raise the failure" do
          expect { instance.stop }.to raise_error(HashiCorp::VagrantVMwareDesktop::Errors::VMRunError)
        end
      end

      context "when hard stop fails and guest is not running" do
        before do
          expect(instance).to receive(:read_state).and_return(:not_running)
          expect(instance).to receive(:vmrun).with("stop", anything, "hard", any_args).
            and_raise(HashiCorp::VagrantVMwareDesktop::Errors::VMRunError)
        end

        it "should not raise a failure" do
          expect { instance.stop }.not_to raise_error
        end
      end
    end
  end

  describe "#set_vmware_info" do
    let(:response) {
      HashiCorp::VagrantVMwareDesktop::Helper::VagrantUtility::Response.new(response_data)
    }
    let(:response_data) {
      {content: content, success: success}
    }
    let(:content) { {} }
    let(:success) { true }

    context "when request results in an error" do
      let(:message) { double("message") }
      let(:content) { {message: message} }
      let(:success) { false }

      it "should return error" do
        allow(instance).to receive_message_chain(:vagrant_utility, :get).and_return(response)
        expect { instance.send(:set_vmware_info) }.
          to raise_error(HashiCorp::VagrantVMwareDesktop::Errors::DriverAPIVMwareVersionDetectionError)
      end
    end

    context "when license values are set" do
      let(:license) { "value" }
      let(:content) { {license: license} }

      it "should result in standard license with unknown value" do
        allow(instance).to receive_message_chain(:vagrant_utility, :get).and_return(response)
        instance.send(:set_vmware_info)

        expect(instance.standard?).to be_truthy
        expect(instance.professional?).to be_falsey
      end

      context "when license is known fusion" do
        let(:license) { "fusion.pro" }

        it "should result in professional license" do
          allow(instance).to receive_message_chain(:vagrant_utility, :get).and_return(response)
          instance.send(:set_vmware_info)

          expect(instance.professional?).to be_truthy
          expect(instance.standard?).to be_falsey
        end
      end

      context "when license is known workstation" do
        let(:license) { "workstation" }

        it "should result in professional license" do
          allow(instance).to receive_message_chain(:vagrant_utility, :get).and_return(response)
          instance.send(:set_vmware_info)

          expect(instance.professional?).to be_truthy
          expect(instance.standard?).to be_falsey
        end
      end

      context "when license is known combo application standard" do
        let(:license) { "fusion.ws.pro.vl" }

        it "should result in standard license" do
          allow(instance).to receive_message_chain(:vagrant_utility, :get).and_return(response)
          instance.send(:set_vmware_info)

          expect(instance.standard?).to be_truthy
          expect(instance.professional?).to be_falsey
        end
      end
    end

    describe "#get_disks" do
      let(:vmx_contents) do
        "random0:0.maprootshare = \"TRUE\"\n" \
        "ide0:0.devicetype = \"cdrom-raw\"\n" \
        "ide0:0.filename = \"auto detect\"\n" \
        "ide0:0.present = \"TRUE\"\n" \
        "scsi0.present = \"TRUE\"\n" \
        "scsi0:0.filename = \"disk-000018.vmdk\"\n" \
        "scsi0:0.present = \"TRUE\"\n" \
        "scsi0:1.filename = \"another_one.vmdk\"\n" \
        "scsi0:1.present = \"TRUE\"\n"
      end

      it "returns all the devices of given type" do
        expected = {
          "random0:0"=>{"maprootshare"=>"TRUE"},
          "scsi0:0"=>{"filename"=>"disk-000018.vmdk", "present"=>"TRUE"},
          "scsi0:1"=>{"filename"=>"another_one.vmdk", "present"=>"TRUE"}
        }
        expect(instance.get_disks(["random", "scsi"])).to eq(expected)
      end
    end

    describe "#remove_disk" do
      let(:vmx) { double("vmx") }

      let(:vmx_contents) do
        "random0:0.maprootshare = \"TRUE\"\n" \
        "ide0:0.devicetype = \"cdrom-raw\"\n" \
        "ide0:0.filename = \"auto detect\"\n" \
        "ide0:0.present = \"TRUE\"\n" \
        "scsi0.present = \"TRUE\"\n" \
        "scsi0:0.filename = \"disk-000018.vmdk\"\n" \
        "scsi0:0.present = \"TRUE\"\n" \
        "scsi0:1.filename = \"another_one.vmdk\"\n" \
        "scsi0:1.present = \"TRUE\"\n"
      end

      before do
        allow(vmx).to receive(:[]=)
        allow(vmx).to receive(:keys).and_return([])
        allow(instance).to receive(:vmx_modify).and_yield(vmx)
      end

      it "removes a disk" do
        allow(File).to receive(:exist?).and_return(true)
        expect(instance).to receive(:vdiskmanager)
        expect(vmx).to receive(:delete).with("scsi0:1.filename")
        expect(vmx).to receive(:delete).with("scsi0:1.present")
        instance.remove_disk("another_one.vmdk")
      end

      it "gracefully handles non existent disk" do
        allow(File).to receive(:exist?).and_return(false)
        expect(instance).not_to receive(:vdiskmanager)
        expect(vmx).not_to receive(:delete)
        instance.remove_disk("oops.vmdk")
      end
    end

    describe "#get_disk_size" do
      before do
        allow(File).to receive(:exist?).and_return(true)
      end

      it "extracts disk size" do
        allow(File).to receive(:foreach)
          .and_yield("createType=\"monolithicSparse\"\n")
          .and_yield("\n")
          .and_yield("# Extent description\n")
          .and_yield("RW 4194304 SPARSE \"another_one.vmdk\"\n")
          .and_yield("\n")
          .and_yield("# The Disk Data Base\n")
        expect(instance.get_disk_size("/some/path.vmdk")).to eq(2147483648)
      end

      it "sums disks size" do
        allow(File).to receive(:foreach)
          .and_yield("createType=\"monolithicSparse\"\n")
          .and_yield("\n")
          .and_yield("# Extent description\n")
          .and_yield("RW 4194304 SPARSE \"another_one.vmdk\"\n")
          .and_yield("RW 4194304 SPARSE \"another_one.vmdk\"\n")
          .and_yield("RW 4194304 SPARSE \"another_one.vmdk\"\n")
          .and_yield("\n")
          .and_yield("# The Disk Data Base\n")
        expect(instance.get_disk_size("/some/path.vmdk")).to eq(6442450944)
      end

      it "gracefully handles non existent disk" do
        allow(File).to receive(:exist?).and_return(false)
        expect(instance.get_disk_size("/some/path/doesnt/exist.vmdk")).to eq(nil)
      end
    end

    describe "#add_disk_to_vmx" do
      let(:vmx) { double("vmx") }

      before do
        allow(vmx).to receive(:[]=)
        allow(vmx).to receive(:keys).and_return([])
        allow(instance).to receive(:vmx_modify).and_yield(vmx)
      end

      it "adds config to vmx file" do
        expect(vmx).to receive(:[]=).with("ide0.present", "TRUE")
        expect(vmx).to receive(:[]=).with("ide0:1.foo", "bar")
        expect(vmx).to receive(:[]=).with("ide0:1.baz", "goo")
        expect(vmx).to receive(:[]=).with("ide0:1.filename", "/some/file.txt")
        expect(vmx).to receive(:[]=).with("ide0:1.present", "TRUE")
        instance.add_disk_to_vmx("/some/file.txt", "ide0:1", {"foo"=>"bar", "baz"=>"goo"})
      end
    end

    describe "#remove_disk_from_vmx" do
      let(:vmx) { double("vmx") }

      let(:vmx_contents) do
        "ide0:1.filename = \"/some/file.txt\"\n"
      end

      before do
        allow(vmx).to receive(:[]=)
        allow(vmx).to receive(:keys).and_return([])
        allow(instance).to receive(:vmx_modify).and_yield(vmx)
      end

      it "adds config to vmx file" do
        expect(vmx).to receive(:delete).with("ide0:1.foo")
        expect(vmx).to receive(:delete).with("ide0:1.baz")
        expect(vmx).to receive(:delete).with("ide0:1.filename")
        expect(vmx).to receive(:delete).with("ide0:1.present")
        instance.remove_disk_from_vmx("/some/file.txt", ["foo", "baz"])
      end
    end

    describe "#snapshot_tree" do
      let(:vmx) { double("vmx") }

      context "with a simple hierarchy of snapshots" do
        let(:vmrun_result){ double(stdout: """Total snapshots: 10
Snapshot
\tSnapshot 2
\t\tSnapshot 3
""") }
        before do
          expect(instance).to receive(:vmrun).with("listSnapshots", vmx_file.path, "showTree").and_return(vmrun_result)
        end

        it "builds a snapshot tree" do
          result = instance.snapshot_tree
          expected_result = ["Snapshot", "Snapshot/Snapshot2", "Snapshot/Snapshot2/Snapshot3",]
          expect(result == expected_result).to be_truthy
        end
      end
     
      context "with a complicated hierarchy of snapshots" do
        let(:vmrun_result){ double(stdout: """Total snapshots: 10
Snapshot
\tSnapshot 2
\t\tSnapshot 3
\t\t\tSnapshot 6
\t\tSnapshot 4
\tSnapshot 5
\t\tSnapshot 7
\t\t\tSnapshot 8
\t\t\t\tSnapshot 10
\tSnapshot 9""") }
        before do
          expect(instance).to receive(:vmrun).with("listSnapshots", vmx_file.path, "showTree").and_return(vmrun_result)
        end

        it "builds a snapshot tree" do
          result = instance.snapshot_tree
          expected_result = ["Snapshot", "Snapshot/Snapshot2", "Snapshot/Snapshot2/Snapshot3",
            "Snapshot/Snapshot2/Snapshot3/Snapshot6", "Snapshot/Snapshot2/Snapshot4", 
            "Snapshot/Snapshot5", "Snapshot/Snapshot5/Snapshot7", "Snapshot/Snapshot5/Snapshot7/Snapshot8",
            "Snapshot/Snapshot5/Snapshot7/Snapshot8/Snapshot10", "Snapshot/Snapshot9"
          ]
          expect(result == expected_result).to be_truthy
        end
      end
    end
  end
end
