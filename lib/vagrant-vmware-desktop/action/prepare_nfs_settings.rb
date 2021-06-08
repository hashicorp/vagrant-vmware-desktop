module HashiCorp
  module VagrantVMwareDesktop
    module Action
      class PrepareNFSSettings
        include Common

        def initialize(app,env)
          @app = app
          @logger = Log4r::Logger.new("vagrant::action::vm::nfs")
        end

        def call(env)
          @app.call(env)

          env[:nfs_machine_ip] = env[:machine].provider.driver.read_ip(
            env[:machine].provider_config.enable_vmrun_ip_lookup
          )

          # We just assume the machine IP is the first 3 octets
          # with "1" for the last octet. Poor assumption but can
          # be fixed later.
          if env[:nfs_machine_ip]
            host_ip = env[:nfs_machine_ip].split(".").map(&:to_i)
            host_ip[3] = 1
            env[:nfs_host_ip] = host_ip.join(".")
          end

          using_nfs = false
          env[:machine].config.vm.synced_folders.each do |id, opts|
            if opts[:nfs]
              using_nfs = true
              break
            end
          end

          if using_nfs
            raise Errors::NFSNoNetwork if !env[:nfs_machine_ip]
          end
        end
      end
    end
  end
end
