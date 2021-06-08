namespace :build do
  desc "Set the state to build for Fusion."
  task :fusion do
    ENV["VAGRANT_VMWARE_PRODUCT"] = "fusion"
  end

  desc "Set the state to build for Workstation."
  task :workstation do
    ENV["VAGRANT_VMWARE_PRODUCT"] = "workstation"
  end

  desc "Build the gem."
  task :gem do
    exec("bash build/build.sh")
  end
end
