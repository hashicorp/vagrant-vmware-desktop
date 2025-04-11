Gem::Specification.new do |s|
  s.name                 = "vagrant-vmware-desktop"
  s.version              = File.read(File.expand_path("../versions/desktop.txt", __FILE__))
  s.platform             = Gem::Platform::RUBY
  s.authors              = "Vagrant Team"
  s.email                = "vagrant@hashicorp.com"
  s.homepage             = "http://www.vagrantup.com"
  s.license              = "MPL-2.0"
  s.summary              = "Enables Vagrant to power VMware Workstation/Fusion machines."
  s.description          = "Enables Vagrant to power VMware Workstation/Fusion machines."
  s.post_install_message = <<-EOF
Thank you for installing the Vagrant VMware Desktop
plugin. This plugin requires the Vagrant VMware
Utility to be installed. To learn more about the
Vagrant VMware Utility, please visit:

  https://www.vagrantup.com/docs/providers/vmware/vagrant-vmware-utility

To install the Vagrant VMware Utility, please
download the appropriate installer for your
system from:

  https://www.vagrantup.com/downloads/vmware
EOF

  s.required_rubygems_version = ">= 1.3.6"

  root_path      = File.dirname(__FILE__)
  all_files      = Dir.chdir(root_path) { Dir.glob("lib/**/*") }
  all_files.concat(Dir.chdir(root_path) { Dir.glob("locales/**/*") })
  all_files.reject! { |file| file.start_with?("lib/hashicorp") }
  all_files.reject! { |file| [".", ".."].include?(File.basename(file)) }

  gitignore_path = File.join(root_path, ".gitignore")
  gitignore      = File.readlines(gitignore_path)
  gitignore.map!    { |line| line.chomp.strip }
  gitignore.reject! { |line| line.empty? || line =~ /^(#|!)/ }

  unignored_files = all_files.reject do |file|
    next true if File.directory?(file)
    gitignore.any? do |ignore|
      File.fnmatch(ignore, file, File::FNM_PATHNAME) ||
        File.fnmatch(ignore, File.basename(file), File::FNM_PATHNAME)
    end
  end

  s.files         = unignored_files
  s.executables   = []
  s.require_path  = 'lib'
end
