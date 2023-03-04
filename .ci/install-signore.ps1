# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

# Download signore release artifact and unzip in the current directory

# Start with getting the download URL for the latest
# signore release
$request = @{
    Uri = "https://api.github.com/repos/hashicorp/signore/releases/latest"
    Headers = @{
        Accept = "application/vnd.github+json"
        Authentication = "token ${env:HASHIBOT_TOKEN}"
    }
}

try {
    $response = Invoke-WebRequest @request
} catch  {
    Write-Error "Request for latest signore release failed: ${PSItem}"
    exit 1
}

if ($response.StatusCode -ne 200) {
    Write-Error "Request for latest signore release returned unexpected status code: ${response.StatusCode} != 200"
    exit 1
}

$artifactUrl = $response.Content | jq ".[] | select(.name | contains(`"`"`"windows_x86_64.zip"`"`"`)) | .url"

if ($artifactUrl -eq "") {
    Write-Error "Failed to detect latest signore release for Windows (empty match)"
    exit 1
}

# Download signore release artifact
$request = @{
    Uri = $artifactUrl
    Headers = @{
        Authentication = "token ${env:HASHIBOT_TOKEN}"
    }
    OutFile = "signore.zip"
}

try {
    $response = Invoke-WebRequest @request
} catch {
    Write-Error "Request for latest signore release artifact failed: ${PSItem}"
}

if ($response.StatusCode -ne 200) {
    Write-Error "Request for latest signore release artifact returned unexpected status code: ${response.StatusCode} != 200"
    exit 1
}

try {
    Expand-Archive -Path signore.zip -DestinationPath .
} catch {
    Write-Error "Expansion of signore release artifact failed: ${PSItem}"
    exit 1
}
