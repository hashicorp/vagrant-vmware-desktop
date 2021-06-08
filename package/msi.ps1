<#
.SYNOPSIS
    Packages a Vagrant VMware Utility installer.

.DESCRIPTION
    Packages a Vagrant VMware Utility installer.

    This script requires administrative privileges.

    You can run this script from an old-style cmd.exe prompt using the
    following:

      powershell.exe -ExecutionPolicy Unrestricted -NoLogo -NoProfile -Command "& '.\msi.ps1'"

#>

param(
    [string]$SignKey="",
    [string]$SignKeyPassword="",
    [string]$SignPath=""
)


# Exit if there are any exceptions
$ErrorActionPreference = "Stop"

# Put this in a variable to make things easy later
$UpgradeCode = "e43b0f5f-e2fe-430a-9ac9-969860928b4a"

# Get the directory to this script
$Dir = Split-Path $script:MyInvocation.MyCommand.Path

# Lookup the WiX binaries, these will error if they're not on the Path
$WixHeat   = Get-Command heat | Select-Object -ExpandProperty Definition
$WixCandle = Get-Command candle | Select-Object -ExpandProperty Definition
$WixLight  = Get-Command light | Select-Object -ExpandProperty Definition

Write-Host "==> Setting up for msi build..."

$package = $Dir
$root = Resolve-Path -Path "${package}\.."
$base = "${root}\pkg"
$binaries = "${base}\binaries"
$stage = [System.IO.Path]::GetTempPath()
$stage = [System.IO.Path]::Combine($stage, [System.IO.Path]::GetRandomFileName())
[System.IO.Directory]::CreateDirectory("${stage}\bin") | Out-Null
$utility_path = "${binaries}\vagrant-vmware-utility_windows_amd64.exe"

Write-Host "==> Installing vagrant-vmware-utility..."

Copy-Item "${utility_path}" -Destination "${stage}\bin\vagrant-vmware-utility.exe"

$version = (cmd /c "${stage}\bin\vagrant-vmware-utility.exe --version" 2`>`&1)

Write-Host "==> Detecting utility version... ${version}!"

$asset = "${base}\vagrant-vmware-utility_${version}_x86_64.msi"

$InstallerTmpDir = [System.IO.Path]::GetTempPath()
$InstallerTmpDir = [System.IO.Path]::Combine(
    $InstallerTmpDir, [System.IO.Path]::GetRandomFileName())
[System.IO.Directory]::CreateDirectory($InstallerTmpDir) | Out-Null
[System.IO.Directory]::CreateDirectory("${InstallerTmpDir}\assets") | Out-Null

Copy-Item "${package}\msi\bg_banner.bmp" `
    -Destination "${InstallerTmpDir}\assets\bg_banner.bmp"
Copy-Item "${package}\msi\bg_dialog.bmp" `
    -Destination "${InstallerTmpDir}\assets\bg_dialog.bmp"
Copy-Item "${package}\msi\license.rtf" `
    -Destination "${InstallerTmpDir}\assets\license.rtf"
Copy-Item "${package}\msi\burn_logo.bmp" `
    -Destination "${InstallerTmpDir}\assets\burn_logo.bmp"
Copy-Item "${package}\msi\vagrant.ico" `
    -Destination "${InstallerTmpDir}\assets\vagrant.ico"
Copy-Item "${package}\msi\vagrant-vmware-utility-en-us.wxl" `
    -Destination "${InstallerTmpDir}\vagrant-vmware-utility-en-us.wxl"

$contents = @"
<?xml version="1.0" encoding="utf-8"?>
<Include>
  <?define VersionNumber="${version}" ?>
  <?define DisplayVersionNumber="${version}" ?>

  <!--
    Upgrade code must be unique per version installer.
    This is used to determine uninstall/reinstall cases.
  -->
  <?define UpgradeCode="${UpgradeCode}" ?>
</Include>
"@
$contents | Out-File `
    -Encoding ASCII `
    -FilePath "${InstallerTmpDir}\vagrant-config.wxi"

$contents = @"
<?xml version="1.0"?>
<Wix xmlns="http://schemas.microsoft.com/wix/2006/wi" xmlns:util="http://schemas.microsoft.com/wix/UtilExtension">
  <!-- Include our wxi -->
  <?include "${InstallerTmpDir}\vagrant-config.wxi" ?>

  <!-- The main product -->
  <Product Id="*"
           Language="!(loc.LANG)"
           Name="!(loc.ProductName)"
           Version="`$(var.VersionNumber)"
           Manufacturer="!(loc.ManufacturerName)"
           UpgradeCode="`$(var.UpgradeCode)">

    <!-- Define the package information -->
    <Package Compressed="yes"
             InstallerVersion="200"
             InstallPrivileges="elevated"
             InstallScope="perMachine"
             Manufacturer="!(loc.ManufacturerName)" />

    <!-- Disallow installing older versions until the new version is removed -->
    <!-- Note that this creates the RemoveExistingProducts action -->
    <MajorUpgrade DowngradeErrorMessage="A later version of Vagrant VMware Utility is installed. Please remove this version first. Setup will now exit."
                  Schedule="afterInstallInitialize" />

    <!-- Check that VMware Workstation is installed -->
    <Property Id="VMWAREINSTALLED">
      <RegistrySearch Id="VMwareInstallSearch" Root="HKLM" Key="SOFTWARE\VMware, Inc.\VMware Workstation" Name="InstallPath" Type="raw" Win64="no" />
    </Property>
    <Condition Message="Vagrant VMware Utility requires a valid installation of VMware Workstation. Please install VMware Workstation and then run this installer again.">
      VMWAREINSTALLED OR Installed
    </Condition>

    <!-- The source media for the installer -->
    <Media Id="1"
           Cabinet="VagrantVMwareUtility.cab"
           CompressionLevel="high"
           EmbedCab="yes" />

     <!-- Require Windows NT Kernel -->
     <Condition Message="This application is only supported on Windows 2000 or higher.">
       <![CDATA[Installed or (VersionNT >= 500)]]>
     </Condition>

     <!-- Include application icon for add/remove programs -->
     <Icon Id="icon.ico" SourceFile="$($InstallerTmpDir)\assets\vagrant.ico" />
     <Property Id="ARPPRODUCTICON" Value="icon.ico" />
     <Property Id="ARPHELPLINK" Value="https://www.vagrantup.com" />

     <!-- Get the proper system directory -->
     <SetDirectory Id="WINDOWSVOLUME" Value="[WindowsVolume]" />

     <PropertyRef Id="WIX_ACCOUNT_USERS" />
     <PropertyRef Id="WIX_ACCOUNT_ADMINISTRATORS" />

     <!-- The directory where we'll install Vagrant -->
     <Directory Id="TARGETDIR" Name="SourceDir">
       <Directory Id="WINDOWSVOLUME">
         <Directory Id="MANUFACTURERDIR" Name="HashiCorp">
           <Directory Id="INSTALLDIR" Name="VagrantVMwareUtility">
             <Component Id="VagrantBin"
               Guid="{05B947B5-7A8F-4AA1-9B76-A7844BF21BD4}">

               <!-- Because we are not in "Program Files" we inherit
                    permissions that are not desirable. Force new permissions -->
               <CreateFolder>
                 <Permission GenericAll="yes" User="[WIX_ACCOUNT_ADMINISTRATORS]" />
                 <Permission GenericRead="yes" GenericExecute="yes" User="[WIX_ACCOUNT_USERS]" />
               </CreateFolder>
             </Component>
           </Directory>
         </Directory>
       </Directory>
     </Directory>

     <!-- Define the features of our install -->
     <Feature Id="VagrantFeature"
              Title="!(loc.ProductName)"
              Level="1">
       <ComponentGroupRef Id="VagrantVMwareUtilityDir" />
       <ComponentRef Id="VagrantBin" />
     </Feature>

     <!-- WixUI configuration so we can have a UI -->
     <Property Id="WIXUI_INSTALLDIR" Value="INSTALLDIR" />

     <UIRef Id="VagrantUI_InstallDir" />
     <UI Id="VagrantUI_InstallDir">
       <UIRef Id="WixUI_InstallDir" />
     </UI>

     <WixVariable Id="WixUILicenseRtf" Value="$($InstallerTmpDir)\assets\license.rtf" />
     <WixVariable Id="WixUIDialogBmp" Value="$($InstallerTmpDir)\assets\bg_dialog.bmp" />
     <WixVariable Id="WixUIBannerBmp" Value="$($InstallerTmpDir)\assets\bg_banner.bmp" />
     <!-- Install the Utility Windows service -->
     <CustomAction Id="UtilityCertificateInstall"
                   FileKey="filAD403B2949D17CD1DA10C7CA66DB5D96"
                   ExeCommand="certificate generate"
                   Execute="deferred"
                   Impersonate="no"
                   Return="ignore" />
     <CustomAction Id="UtilityServiceInstall"
                   FileKey="filAD403B2949D17CD1DA10C7CA66DB5D96"
                   ExeCommand="service install"
                   Execute="deferred"
                   Impersonate="no"
                   Return="ignore" />
     <CustomAction Id="UtilityServiceUninstall"
                   FileKey="filAD403B2949D17CD1DA10C7CA66DB5D96"
                   ExeCommand="service uninstall"
                   Execute="deferred"
                   Impersonate="no"
                   Return="ignore" />
    <InstallExecuteSequence>
      <Custom Action="UtilityCertificateInstall" Before="InstallFinalize" />
      <Custom Action="UtilityServiceInstall" Before="InstallFinalize" />
      <Custom Action="UtilityServiceUninstall" Before="RemoveFiles">
        (NOT UPGRADINGPRODUCTCODE) AND (REMOVE="ALL")
      </Custom>
    </InstallExecuteSequence>
  </Product>
</Wix>
"@
$contents | Out-File `
    -Encoding ASCII `
    -FilePath "${InstallerTmpDir}\vagrant-main.wxs"

Write-Host "Running heat.exe"
& $WixHeat dir $stage `
    -nologo `
    -ke `
    -sreg `
    -srd `
    -gg `
    -cg VagrantVMwareUtilityDir `
    -dr INSTALLDIR `
    -var 'var.VagrantSourceDir' `
    -out "${InstallerTmpDir}\vagrant-files.wxs"

if(!$?) {
    Write-Host "Error: Failed running heat.exe"
    exit 1
}

Write-Host "Running candle.exe"
$CandleArgs = @(
    "-nologo",
    "-arch x64",
    "-I${InstallerTmpDir}",
    "-dVagrantSourceDir=${stage}",
    "-out $InstallerTmpDir\",
    "${InstallerTmpDir}\vagrant-files.wxs",
    "${InstallerTmpDir}\vagrant-main.wxs"
)
Start-Process -NoNewWindow -Wait `
    -ArgumentList $CandleArgs -FilePath $WixCandle

if(!$?) {
    Write-Host "Error: Failed running candle.exe"
    exit 1
}

Write-Host "Running light.exe"
& $WixLight `
    -nologo `
    -ext WixUIExtension `
    -ext WixUtilExtension `
    -spdb `
    -v `
    -cultures:en-us `
    -loc "${InstallerTmpDir}\vagrant-vmware-utility-en-us.wxl" `
    -out $asset `
    "${InstallerTmpDir}\vagrant-files.wixobj" `
    "${InstallerTmpDir}\vagrant-main.wixobj"

if(!$?) {
    Write-Host "Error: Failed running light.exe"
    exit 1
}

#--------------------------------------------------------------------
# Sign
#--------------------------------------------------------------------
if ("${SignKey}" -ne "") {
    $SignKeyExists = Test-Path -LiteralPath $SignKey
}

if ($SignKeyExists -and "${SignKeyPassword}" -ne "") {
    Write-Host "==> Signing installer package asset..."
    $SignTool = "signtool.exe"
    if ($SignPath) {
        $SignTool = $SignPath
    }

    Write-Host "!!> Path used for signtool... (${SignTool})"
    Write-Host "!!> Applying signature..."

    & $SignTool sign `
      /debug `
      /t http://timestamp.digicert.com `
      /f "${SignKey}" `
      /p "${SignKeyPassword}" `
      "${asset}"

    if(!$?) {
        Write-Host "Error: Failed signing installer package"
        exit 1
    }

    Write-Host "  **> Installer package asset is signed <** "
} else {
    Write-Host ""
    Write-Host "!!!!!!!!!!!! WARNING !!!!!!!!!!!!!!!"
    Write-Host "! Vagrant VMware Utility installer !"
    Write-Host "! package is NOT signed. Rebuild   !"
    Write-Host "! with signing key for release     !"
    Write-Host "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!"
    Write-Host ""
}

Write-Host "==> Cleaning up packaging artifacts..."
Remove-Item -Recurse -Force $InstallerTmpDir
Remove-Item -Recurse -Force $stage

Write-Host "==> Package build complete: ${asset}"
