param(
    [string]$OutputDirectory = "dist"
)

$ErrorActionPreference = "Stop"
$projectRoot = Split-Path -Parent $PSScriptRoot
$distPath = Join-Path $projectRoot $OutputDirectory
$requiredFiles = @("config.example.json", "LICENSE", "README.md", "README.ko.md")

foreach ($file in $requiredFiles) {
    if (-not (Test-Path (Join-Path $projectRoot $file))) {
        throw "Required release file is missing: $file"
    }
}

New-Item -ItemType Directory -Force -Path $distPath | Out-Null

$targets = @(
    @{ Name = "beszelview_windows_amd64"; GOOS = "windows"; GOARCH = "amd64"; Binary = "beszelview.exe"; Format = "zip" },
    @{ Name = "beszelview_linux_amd64"; GOOS = "linux"; GOARCH = "amd64"; Binary = "beszelview"; Format = "tar.gz" },
    @{ Name = "beszelview_linux_arm"; GOOS = "linux"; GOARCH = "arm"; GOARM = "7"; Binary = "beszelview"; Format = "tar.gz" },
    @{ Name = "beszelview_linux_arm64"; GOOS = "linux"; GOARCH = "arm64"; Binary = "beszelview"; Format = "tar.gz" }
)

$originalGOOS = $env:GOOS
$originalGOARCH = $env:GOARCH
$originalGOARM = $env:GOARM
$originalCGO = $env:CGO_ENABLED

try {
    $env:CGO_ENABLED = "0"
    foreach ($target in $targets) {
        $stage = Join-Path $distPath $target.Name
        if (Test-Path $stage) {
            Remove-Item -Recurse -Force -LiteralPath $stage
        }
        New-Item -ItemType Directory -Path $stage | Out-Null

        $env:GOOS = $target.GOOS
        $env:GOARCH = $target.GOARCH
        $env:GOARM = if ($target.ContainsKey("GOARM")) { $target.GOARM } else { "" }

        $binaryPath = Join-Path $stage $target.Binary
        & go build -buildvcs=false -trimpath -ldflags "-s -w" -o $binaryPath ./cmd/beszel-monitor
        if ($LASTEXITCODE -ne 0) {
            throw "Build failed for $($target.Name)"
        }

        foreach ($file in $requiredFiles) {
            Copy-Item -LiteralPath (Join-Path $projectRoot $file) -Destination $stage
        }

        if ($target.Format -eq "zip") {
            $archive = Join-Path $distPath ($target.Name + ".zip")
            if (Test-Path $archive) { Remove-Item -LiteralPath $archive }
            Compress-Archive -Path (Join-Path $stage "*") -DestinationPath $archive
        } else {
            $archive = Join-Path $distPath ($target.Name + ".tar.gz")
            if (Test-Path $archive) { Remove-Item -LiteralPath $archive }
            & tar -czf $archive -C $stage $target.Binary @requiredFiles
            if ($LASTEXITCODE -ne 0) {
                throw "Archive creation failed for $($target.Name)"
            }
        }

        Remove-Item -Recurse -Force -LiteralPath $stage
        Write-Host "Created $archive"
    }
} finally {
    $env:GOOS = $originalGOOS
    $env:GOARCH = $originalGOARCH
    $env:GOARM = $originalGOARM
    $env:CGO_ENABLED = $originalCGO
}

