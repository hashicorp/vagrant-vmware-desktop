# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

require "vagrant-vmware-desktop/helper/routing_table"

describe HashiCorp::VagrantVMwareDesktop::Helper::RoutingTable do
  # This fakes the result of a subprocess call for the routing
  # table stuff.
  def setup_results(exit_code, stdout="", stderr="")
    allow(Vagrant::Util::Which).to receive(:which).and_return("netstat")

    result = double("netstat execute")
    allow(Vagrant::Util::Subprocess).to receive(:execute).and_return(result)

    allow(result).to receive_messages(:exit_code => exit_code)
    allow(result).to receive_messages(:stdout => stdout)
    allow(result).to receive_messages(:stderr => stderr)
  end

  def stub_platform(platform)
    darwin = platform == :darwin
    linux  = platform == :linux
    windows = platform == :windows

    allow(Vagrant::Util::Platform).to receive_messages(:darwin? => darwin)
    allow(Vagrant::Util::Platform).to receive_messages(:linux? => linux)
    allow(Vagrant::Util::Platform).to receive_messages(:windows? => windows)
  end

  context "darwin" do
    before :each do
      stub_platform(:darwin)
    end

    it "should raise an error if netstat is not on path" do
      allow(Vagrant::Util::Which).to receive(:which).and_return(nil)

      expect { described_class.new }.
        to raise_error(HashiCorp::VagrantVMwareDesktop::Errors::RoutingTableCommandNotFound)
    end

    it "should raise an error if netstat fails (exit code != 0)" do
      setup_results(1)

      expect { described_class.new }.
        to raise_error(HashiCorp::VagrantVMwareDesktop::Errors::RoutingTableLoadError)
    end

    context "device_for_route" do
      before(:each) do
        setup_results(0, <<-STDOUT, "")
Routing tables

Internet:
Destination        Gateway            Flags        Refs      Use   Netif Expire
0/1                10.137.0.9         UGSc           66        0   utun0
default            10.0.1.1           UGSc           26        0     en0
10.0.1/24          link#4             UCS             4        0     en0
10.0.1.1           b8:c7:5d:ce:8f:6b  UHLWIir        27     3395     en0    702
10.0.1.32          1c:ab:a7:a1:54:ac  UHLWIi          0     2782     en0    822
10.0.1.41          127.0.0.1          UHS             0        0     lo0
10.0.1.48          b8:c7:5d:ce:8f:6b  UHLWIi          0        0     en0   1176
10.0.1.255         ff:ff:ff:ff:ff:ff  UHLWbI          0       21     en0
30.30.30/24        link#7             UC              2        0  vmnet2
30.30.30.255       ff:ff:ff:ff:ff:ff  UHLWbI          0       21  vmnet2
33.33.33/24        link#8             UC              2        0  vmnet3
33.33.33.255       ff:ff:ff:ff:ff:ff  UHLWbI          0       21  vmnet3
127                127.0.0.1          UCS             0        0     lo0
127.0.0.1          127.0.0.1          UH              7    71930     lo0
169.254            link#4             UCS             0        0     en0
172.16.163/24      link#9             UC              3        0  vmnet8
172.16.163.255     ff:ff:ff:ff:ff:ff  UHLWbI          0       21  vmnet8
192.168.51         link#6             UC              2        0  vmnet1
192.168.51.1       0:50:56:c0:0:1     UHLWIi          1       12     lo0
192.168.51.255     ff:ff:ff:ff:ff:ff  UHLWbI          0       21  vmnet1
192.168.1          link#4             UCS             0        0     en1
192.168.102        link#4             UCS             0        0  vmnet4
        STDOUT
      end

      it "should route prefixes" do
        expect(subject.device_for_route("127.1.2.3")).to eq("lo0")
      end

      it "should route exact matches" do
        expect(subject.device_for_route("10.0.1.48")).to eq("en0")
      end

      it "should route CIDR matches" do
        expect(subject.device_for_route("33.33.33.10")).to eq("vmnet3")
        expect(subject.device_for_route("33.33.33.0")).to eq("vmnet3")
      end

      it "should return nil if there is no route" do
        expect(subject.device_for_route("255.255.255.255")).to be_nil
      end

      it "should match prefixes to full octets" do
        expect(subject.device_for_route("192.168.102.1")).to eq("vmnet4")
      end
    end
  end

  context "linux" do
    before :each do
      stub_platform(:linux)
    end

    it "should raise an error if netstat is not on path" do
      allow(Vagrant::Util::Which).to receive(:which).and_return(nil)

      expect { described_class.new }.
        to raise_error(HashiCorp::VagrantVMwareDesktop::Errors::RoutingTableCommandNotFound)
    end

    it "should raise an error if netstat fails (exit code != 0)" do
      setup_results(1)

      expect { described_class.new }.
        to raise_error(HashiCorp::VagrantVMwareDesktop::Errors::RoutingTableLoadError)
    end

    describe "#device_for_route" do
      before :each do
        setup_results(0, <<-STDOUT, "")
Kernel IP routing table
Destination     Gateway         Genmask         Flags   MSS Window  irtt Iface
0.0.0.0         178.63.23.65    0.0.0.0         UG        0 0          0 eth0
178.63.23.64    178.63.23.65    255.255.255.192 UG        0 0          0 eth0
178.63.23.64    0.0.0.0         255.255.255.192 U         0 0          0 eth0
192.168.97.0    0.0.0.0         255.255.255.0   U         0 0          0 vmnet1
192.168.122.0   0.0.0.0         255.255.255.0   U         0 0          0 vmnet8
        STDOUT
      end

      it "should route CIDR matches" do
        expect(subject.device_for_route("178.63.23.88")).to eq("eth0")
        expect(subject.device_for_route("192.168.122.131")).to eq("vmnet8")
      end

      it "should return nil if there is no route" do
        expect(subject.device_for_route("33.34.10.10")).to be_nil
      end
    end
  end

  context "windows" do
    before :each do
      stub_platform(:windows)
    end

    it "should raise an error if netsh is not on path" do
      allow(Vagrant::Util::Which).to receive(:which).and_return(nil)

      expect { described_class.new }.
        to raise_error(HashiCorp::VagrantVMwareDesktop::Errors::RoutingTableCommandNotFound)
    end

    it "should raise an error if netsh fails (exit code != 0)" do
      setup_results(1)

      expect { described_class.new }.
        to raise_error(HashiCorp::VagrantVMwareDesktop::Errors::RoutingTableLoadError)
    end

    describe "#device_for_route" do
      before :each do
        setup_results(0, <<-STDOUT, "")

Publish  Type      Met  Prefix                    Idx  Gateway/Interface Name
-------  --------  ---  ------------------------  ---  ------------------------
No       Manual    0    0.0.0.0/0                  13  192.168.1.1
No       System    256  127.0.0.0/8                 1  Loopback Pseudo-Interface 1
No       System    256  127.0.0.1/32                1  Loopback Pseudo-Interface 1
No       System    256  127.255.255.255/32          1  Loopback Pseudo-Interface 1
No       System    256  192.168.1.0/24             13  Wi-Fi
No       System    256  192.168.1.170/32           13  Wi-Fi
No       System    256  192.168.1.255/32           13  Wi-Fi
No       System    256  192.168.222.0/24           23  VMware Network Adapter VMnet8
No       System    256  192.168.222.1/32           23  VMware Network Adapter VMnet8
No       System    256  192.168.222.255/32         23  VMware Network Adapter VMnet8
No       System    256  224.0.0.0/4                 1  Loopback Pseudo-Interface 1
No       System    256  224.0.0.0/4                15  Bluetooth Network Connection
No       System    256  224.0.0.0/4                36  Local Area Connection* 14
No       System    256  224.0.0.0/4                23  VMware Network Adapter VMnet8
No       System    256  224.0.0.0/4                12  Ethernet
No       System    256  224.0.0.0/4                13  Wi-Fi
No       System    256  255.255.255.255/32          1  Loopback Pseudo-Interface 1
No       System    256  255.255.255.255/32         15  Bluetooth Network Connection
No       System    256  255.255.255.255/32         36  Local Area Connection* 14
No       System    256  255.255.255.255/32         23  VMware Network Adapter VMnet8
No       System    256  255.255.255.255/32         12  Ethernet
No       System    256  255.255.255.255/32         13  Wi-Fi

        STDOUT
      end

      it "should route CIDR matches" do
        expect(subject.device_for_route("192.168.1.231")).to eq("Wi-Fi")
      end

      it "should match vmnet devices to just the vmnet name" do
        expect(subject.device_for_route("192.168.222.123")).to eq("vmnet8")
      end

      it "should return nil if there is no route" do
        expect(subject.device_for_route("33.34.10.10")).to be_nil
      end
    end
  end

  context "unknown OS" do
    before :each do
      stub_platform(nil)
    end

    it "should raise an error about unsupported platforms" do
      expect { subject }.
        to raise_error(HashiCorp::VagrantVMwareDesktop::Errors::RoutingTableUnsupportedOS)
    end
  end
end
