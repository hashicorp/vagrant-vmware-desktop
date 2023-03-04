# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

<#
.SYNOPSIS
Bootstrap for package building

.DESCRIPTION
Installs required software, setups up, and builds package
#>

$SignKeyPath = "C:\Users\vagrant\Win_CodeSigning.p12"
$SignKeyExists = Test-Path -LiteralPath $SignKeyPath

$PackageScript = "C:\vagrant\package\msi.ps1"

if($SignKeyExists) {
    if(!$env:SignKeyPassword){
        Write-Host "Error: No password provided for code signing key!"
        exit 1
    }

    $PackageParams = @{
        "SignKey"="${SignKeyPath}";
        "SignKeyPassword"="${env:SignKeyPassword}"
    }
    & $PackageScript @PackageParams
} else {
    & $PackageScript
}

if(!$?) {
    Write-Host "Error: Package failure encountered"
    exit 1
}
