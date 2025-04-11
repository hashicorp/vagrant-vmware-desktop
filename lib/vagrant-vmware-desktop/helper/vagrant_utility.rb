# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

require "log4r"
require "net/http"
require "net/https"

require "vagrant"
require "vagrant/util/downloader"
require "vagrant/util/string_block_editor"

module HashiCorp
  module VagrantVMwareDesktop
    module Helper
      # This is a class for dealing the vagrant vmware utility API
      class VagrantUtility

        # Response wrapper class
        class Response

          # Raw value being wrapped
          #
          # @return [Hash]
          attr_reader :value

          def initialize(value)
            if !value.is_a?(Hash)
              raise TypeError.new("Expecting value of `Hash` type but received `#{value.class}`")
            end
            @value = value
          end

          # Provides Hash#dig functionality but will raise
          # an invalid response exception if given path raises
          # an error.
          #
          # @return [Object]
          def get(*args)
            begin
              value.dig(*args)
            rescue => err
              raise Errors::DriverAPIInvalidResponse
            end
          end

          def [](v)
            value[v]
          end

          # @return [TrueClass, FalseClass] response is success
          def success?
            value[:success]
          end
        end

        # @return [Net::HTTP]
        attr_reader :connection
        # @return [Hash]
        attr_reader :headers
        # @return [Log4r::Logger]
        attr_reader :logger

        def initialize(host, port, **opts)
          @logger = Log4r::Logger.new("hashicorp::provider::vmware::vagrant_utility")
          @logger.debug("initialize HOST=#{host} PORT=#{port}")
          @connection = Net::HTTP.new(host, port)
          @connection.use_ssl = true
          @connection.verify_mode = OpenSSL::SSL::VERIFY_PEER
          @connection.ca_file = File.join(opts[:certificate_path], "vagrant-utility.crt")
          @headers = {
            "Content-Type" => "application/vnd.hashicorp.vagrant.vmware.rest-v1+json",
            "Origin" => "https://#{host}:#{port}",
            "User-Agent" => Vagrant::Util::Downloader::USER_AGENT +
              " - VagrantVMWareDesktop/#{VagrantVMwareDesktop::VERSION}",
            "X-Requested-With" => "Vagrant",
          }
          cert_path = File.join(opts[:certificate_path], "vagrant-utility.client.crt")
          key_path = File.join(opts[:certificate_path], "vagrant-utility.client.key")
          begin
            @connection.cert = OpenSSL::X509::Certificate.new(File.read(cert_path))
          rescue => err
            @logger.debug("certificate load failure - #{err.class}: #{err}")
            raise Errors::DriverAPICertificateError.new(
              path: cert_path,
              message: err.message
            )
          end
          begin
            @connection.key = OpenSSL::PKey::RSA.new(File.read(key_path))
          rescue => err
            @logger.debug("key load failure - #{err.class}: #{err}")
            raise Errors::DriverAPIKeyError.new(
              path: key_path,
              message: err.message
            )
          end
        end

        # Perform GET
        #
        # @param [String] path
        # @return [Response]
        def get(path)
          perform_request(:get, path)
        end

        # Perform PUT
        #
        # @param [String] path
        # @param [Object] payload
        # @return [Response]
        def put(path, payload=nil)
          perform_request(:put, path, payload)
        end

        # Perform POST
        #
        # @param [String] path
        # @param [Object] payload
        # @return [Response]
        def post(path, payload=nil)
          perform_request(:post, path, payload)
        end

        # Perform DELETE
        #
        # @param [String] path
        # @param [Object] payload
        # @return [Response]
        def delete(path, payload=nil)
          perform_request(:delete, path, payload)
        end

        # Perform the remote request and process the result
        #
        # @param [String,Symbol] method HTTP method
        # @param [String] path remote path
        # @param [Object] data request body
        # @param [Hash] rheaders custom request headers
        # @return [Response]
        def perform_request(method, path, data=nil, rheaders={})
          req_headers = headers.merge(rheaders)
          if data && !data.is_a?(String)
            data = JSON.generate(data)
          end
          method = method.to_s.upcase
          response = process_response do
            connection.send_request(method, path, data, req_headers)
          end
          if !response.success?
            error = "ERROR=#{response.get(:content, :message)}"
          end
          @logger.debug("request METHOD=#{method} PATH=#{path} RESPONSE=#{response.get(:code)} #{error}")
          response
        end

        # Wraps response into Response instance
        #
        # @yieldblock [Net::HTTPResponse]
        # @return [Response]
        def process_response
          begin
            response = yield
            result = {
              code: response.code.to_i,
              success: response.code.start_with?('2')
            }
            if response.class.body_permitted?
              body = response.body
              begin
                result[:content] = JSON.parse(body, :symbolize_names => true)
              rescue
                result[:content] = body
              end
            else
              result[:content] = nil
            end
            Response.new(result)
          rescue Net::HTTPServiceUnavailable
            raise Errors::DriverAPIConnectionFailed
          rescue => err
            @logger.debug("unexpected error - #{err.class}: #{err}")
            raise Errors::DriverAPIRequestUnexpectedError, error: err
          end
        end
      end
    end
  end
end
