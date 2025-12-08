# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

<#
.SYNOPSIS
Creates a Vagrant VMware Utility installer.

.DESCRIPTION
Builds a Windows installer package for the Vagrant
VMware utility.

This script requires administrative privileges which
are required by MSI utilities invoked by WiX when running
ICE modules.

.PARAMETER Version
Version of the Vagrant VMware Utility

.PARAMETER UtilityPath
Path to the vagrant-vmware-utility.exe binary

.PARAMETER Destination
Directory to write installer artifact

.PARAMETER DebugDirectory
Optional directory to place generated files for inspection
#>

param(
    [Parameter(Mandatory=$True)]
    [string]$Version,
    [Parameter(Mandatory=$True)]
    [string]$UtilityPath,
    [Parameter(Mandatory=$True)]
    [string]$Destination,
    [Parameter(Mandatory=$False)]
    [string]$DebugDirectory
)

# Make sure WiX is available on the Path
$wixdir = Get-Item -Path "C:\Program Files (x86)\WiX Toolset*"
$wixpath = $wixdir.FullName
$env:Path = "${env:Path};${wixpath}\bin"

# Helper to create a temporary directory
function New-TemporaryDirectory {
    $t = New-TemporaryFile
    Remove-Item -Path $t -Force | Out-Null
    New-Item -ItemType Directory -Path $t.FullName | Out-Null
    Get-Item -Path $t.FullName
}

# Exit if any exceptions are encountered
$ErrorActionPreference = "Stop"

# Directory this script is located within
$BinDirectory = Resolve-Path $PSScriptRoot
# Base package directory
$PackageDirectory = Split-Path -Parent -Path $BinDirectory
# Repository root directory
$ProjectDirectory = Split-Path -Parent -Path $PackageDirectory

# Find the required WiX binaries. If any of these are not
# found it will force an error.
$WiXHeat = Get-Command heat | Select-Object -ExpandProperty Definition
$WiXCandle = Get-Command candle | Select-Object -ExpandProperty Definition
$WiXLight = Get-Command light | Select-Object -ExpandProperty Definition

# Get full path to binary and destination directory
$UtilityPath = Resolve-Path $UtilityPath

# Create the destination directory if it does not exist
if ( ! ($Destination | Test-Path) ) {
    New-Item -ItemType Directory -Path $Destination | Out-Null
}
$Destination = Resolve-Path $Destination

# If DebugDirectory is defined, get full path and flag debug storage
if ( $DebugDirectory -ne "" ) {
    if ( ! ($DebugDirectory | Test-Path) ) {
        New-Item -ItemType Directory -Path $DebugDirectory | Out-Null
    }
    $DebugDirectory = Resolve-Path $DebugDirectory
    $DebugArtifacts = $True
} else {
    $DebugArtifacts = $False
}

# Validate version format and extract version value
if ( ! ($Version -match '(^[0-9]+\.[0-9]+\.[0-9]+)') ) {
    Write-Error "Vagrant VMware Utility version does not look like a valid version (${Version})"
} else {
    $VersionStrict = $Matches[0]
}

# Create a work directory
$WorkDirectory = New-TemporaryDirectory
Push-Location $WorkDirectory

# Create directory for the package structure
$BuildDirectory = New-TemporaryDirectory

# Create directory for the installer structure
$InstallerDirectory = New-TemporaryDirectory

Write-Output "Starting Vagrant VMware Utility installer build"
Write-Output "  Using binary: ${UtilityPath}"
Write-Output "  Using version: ${Version}"
Write-Output "  Using strict version: ${VersionStrict}"
Write-Output "  Package directory: ${PackageDirectory}"
Write-Output "  Project directory: ${ProjectDirectory}"
Write-Output "  Build directory: ${BuildDirectory}"
Write-Output "  Installer directory: ${InstallerDirectory}"
Write-Output "  Work directory: ${WorkDirectory}"
Write-Output ""
Write-Output "-> Package destination: ${Destination}"
Write-Output ""

Write-Output "Creating package structure..."

New-Item -ItemType Directory -Path "${BuildDirectory}\bin" | Out-Null
Copy-Item ${UtilityPath} -Destination "${BuildDirectory}\bin\vagrant-vmware-utility.exe"

Write-Output "Creating installer structure..."

# Copy assets in first
New-Item -ItemType Directory -Path "${InstallerDirectory}\assets" | Out-Null
Copy-Item "${PackageDirectory}\msi\bg_banner.bmp" `
    -Destination "${InstallerDirectory}\assets\bg_banner.bmp"
Copy-Item "${PackageDirectory}\msi\bg_dialog.bmp" `
    -Destination "${InstallerDirectory}\assets\bg_dialog.bmp"
Copy-Item "${PackageDirectory}\msi\license.rtf" `
    -Destination "${InstallerDirectory}\assets\license.rtf"
Copy-Item "${PackageDirectory}\msi\burn_logo.bmp" `
    -Destination "${InstallerDirectory}\assets\burn_logo.bmp"
Copy-Item "${PackageDirectory}\msi\vagrant.ico" `
    -Destination "${InstallerDirectory}\assets\vagrant.ico"
Copy-Item "${PackageDirectory}\msi\vagrant-vmware-utility-en-us.wxl" `
    -Destination "${InstallerDirectory}\vagrant-vmware-utility-en-us.wxl"

# Copy in WiX files
Copy-Item "${PackageDirectory}\msi\vagrant-vmware-utility-en-us.wxl" `
  -Destination "${InstallerDirectory}\vagrant-vmware-utility-en-us.wxl"
Copy-Item "${PackageDirectory}\msi\vagrant-vmware-utility-main.wxs" `
  -Destination "${InstallerDirectory}\vagrant-vmware-utility-main.wxs"

# WiX configuration file needs to be read, updated, and then
# written into the installer path
$ConfigContent = Get-Content -Path "${PackageDirectory}\msi\vagrant-vmware-utility-config.wxi"

# Update configuration variables
# NOTE: The $VersionStrict value is used here because prerelease formats will fail
$ConfigContent = $ConfigContent -replace "%VERSION_NUMBER%",$VersionStrict
$ConfigContent = $ConfigContent -replace "%BASE_DIRECTORY%","${InstallerDirectory}"

# Write the updated config content into the installer path
$ConfigContent | Out-File -Encoding utf8 -FilePath "${InstallerDirectory}\vagrant-vmware-utility-config.wxi"

# Begin the packaging process
Write-Output "Starting the Vagrant VMware Utility packaging process..."

# First run heat.exe against the build directory. This will
# collect information (referred to as "harvesting") about
# all the files (and the directory structure) that are to
# be included within the installer package. This will be
# small given just the utility binary is being distributed.

Write-Output " - Running the path harvest process stage..."

$HeatArgs = @(
    "dir",
    $BuildDirectory,
    "-nologo",
    "-sreg", # Do not harvest registry
    "-srd", # Do not harvest the root directory as an element
    "-gg", # Generate guids during the harvest
    "-g1", # Generate guids without braces
    "-sfrag", # Do not generate directory or component fragments
    "-cg", "VagrantVMwareUtilityDir", # Name of the component group (defined in main.wxs file)
    "-dr", "INSTALLDIR", # Reference name used for the root directory (defined in the main.wxs file)
    "-var", "var.VagrantSourceDir", # Substitute path source with this variable name (used later for replacement)
    "-out", "${InstallerDirectory}\vagrant-vmware-utility-files.wxs"
)

# Launch the heat process
$HeatProc = Start-Process `
  -FilePath $WiXHeat `
  -ArgumentList $HeatArgs `
  -NoNewWindow `
  -PassThru

# Cache the process handle so the ExitCode is populated correctly
$handle = $HeatProc.Handle

# Wait for the process to complete
$HeatProc.WaitForExit()

if ( $HeatProc.ExitCode -ne 0 ) {
    Write-Error "Package process failed during file harvest stage (heat.exe)"
}

# If debug artifacts is enabled, copy out the generated file list
if ( $DebugArtifacts -eq $True ) {
    Copy-Item "${InstallerDirectory}\vagrant-vmware-utility-files.wxs" `
      -Destination "${DebugDirectory}\vagrant-vmware-utility-files.wxs"
}

# The next step is to run the wix files through candle.exe,
# which is the compiler, to generate WiX object files from
# the defined configuration.

Write-Output " - Running the build stage..."

$CandleArgs = @(
    "-nologo",
    "-arch", "x64",
    "-I${InstallerDirectory}", # Defines include directory to search (allows the config.wxi to be located)
    "-dVagrantSourceDir=${BuildDirectory}", # Defines path value for source variable (used in previous heat command)
    "-out", "${InstallerDirectory}\",
    "${InstallerDirectory}\vagrant-vmware-utility-files.wxs",
    "${InstallerDirectory}\vagrant-vmware-utility-main.wxs"
)

# Launch the candle process
$CandleProc = Start-Process `
  -FilePath $WixCandle `
  -ArgumentList $CandleArgs `
  -NoNewWindow `
  -PassThru

# Cache the process handle so the ExitCode is populated correctly
$handle = $CandleProc.Handle

# Wait for the process to complete
$CandleProc.WaitForExit()

if ( $CandleProc.ExitCode -ne 0 ) {
    Write-Error "Package process failed during build stage (candle.exe)"
}

if ( $DebugArtifacts -eq $True ) {
    Copy-Item "${InstallerDirectory}\vagrant-vmware-utility-files.wixobj" `
      -Destination "${DebugDirectory}\vagrant-vmware-utility-files.wixob"
    Copy-Item "${InstallerDirectory}\vagrant-vmware-utility-main.wixobj" `
      -Destination "${DebugDirectory}\vagrant-vmware-utility-main.wixobj"
}

# The final step is to run light.exe, which is the linker, on the
# WiX object files previously generated. The end result will be
# the installer artifact.

Write-Output " - Running the linker stage..."

$LightArgs = @(
    "-nologo",
    "-ext", "WixUIExtension", # Enables the UI extension for installer GUI
    "-ext", "WixUtilExtension" # Enable the util extension used for registry search
    "-spdb", # Disable debugging symbols
    "-v", # Enable verbose output
    "-cultures:en-us", # Localized cultures (currently only en-us is defined)
    "-loc", "${InstallerDirectory}\vagrant-vmware-utility-en-us.wxl", # File to read localized strings from
    "-out", ".\vagrant-vmware-utility.msi",
    "${InstallerDirectory}\vagrant-vmware-utility-files.wixobj",
    "${InstallerDirectory}\vagrant-vmware-utility-main.wixobj"
)

# Launch the light process
$LightProc = Start-Process `
  -FilePath $WiXLight `
  -ArgumentList $LightArgs `
  -NoNewWindow `
  -PassThru

# Cache the process handle so the ExitCode is populated correctly
$handle = $LightProc.Handle

# Wait for the process to complete
$LightProc.WaitForExit()

if ( $LightProc.ExitCode -ne 0 ) {
    Write-Error "Package process failed during linker stage (light.exe)"
}

Write-Output "Vagrant VMware Utility packaging process complete!"

# Move the installer to the destination directory with the
# correctly formatted name

$FinalPath = "${Destination}\vagrant-vmware-utility_${Version}_windows_amd64.msi"

Move-Item -Force -Path .\vagrant-vmware-utility.msi -Destination $FinalPath

Write-Output "  -> ${FinalPath}"
