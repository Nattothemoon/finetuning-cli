# Install the latest `ft` binary from GitHub Releases on Windows.
#
#   iwr -useb https://raw.githubusercontent.com/Nattothemoon/finetuning-cli/main/scripts/install.ps1 | iex
#
# Honors:
#   $env:FT_VERSION       pin a specific tag (default: latest)
#   $env:FT_INSTALL_DIR   override destination (default: %LOCALAPPDATA%\finetuning\bin)
#   $env:FT_REPO          override GitHub repo (default: Nattothemoon/finetuning-cli)

$ErrorActionPreference = 'Stop'

$repo    = if ($env:FT_REPO)    { $env:FT_REPO }    else { 'Nattothemoon/finetuning-cli' }
$version = if ($env:FT_VERSION) { $env:FT_VERSION } else { 'latest' }

$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
    'AMD64' { 'amd64' }
    'ARM64' { 'amd64' } # we don't ship win/arm64 today; amd64 runs under emulation
    default { throw "unsupported architecture: $($env:PROCESSOR_ARCHITECTURE)" }
}

if ($version -eq 'latest') {
    $apiUrl = "https://api.github.com/repos/$repo/releases/latest"
    $release = Invoke-RestMethod -Uri $apiUrl -UseBasicParsing
    $version = $release.tag_name
    if (-not $version) { throw "could not determine latest release of $repo" }
}

$vClean = $version.TrimStart('v')
$archive = "ft_${vClean}_windows_${arch}.zip"
$url = "https://github.com/$repo/releases/download/$version/$archive"

$installDir = if ($env:FT_INSTALL_DIR) { $env:FT_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA 'finetuning\bin' }
New-Item -ItemType Directory -Force -Path $installDir | Out-Null

$tmp = New-Item -ItemType Directory -Path (Join-Path $env:TEMP ("ft-install-" + [guid]::NewGuid()))
try {
    $archivePath = Join-Path $tmp.FullName $archive
    Write-Host "Downloading $url"
    Invoke-WebRequest -Uri $url -OutFile $archivePath -UseBasicParsing
    Expand-Archive -Path $archivePath -DestinationPath $tmp.FullName -Force
    $dest = Join-Path $installDir 'ft.exe'
    Move-Item -Force -Path (Join-Path $tmp.FullName 'ft.exe') -Destination $dest
    Write-Host "Installed ft $version to $dest"

    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    if ($userPath -notlike "*$installDir*") {
        Write-Host "note: $installDir is not on your PATH. Adding it for the current user..."
        [Environment]::SetEnvironmentVariable('Path', "$userPath;$installDir", 'User')
        Write-Host "Open a new terminal for the PATH change to take effect."
    }
    & $dest --version
} finally {
    Remove-Item -Recurse -Force $tmp
}
