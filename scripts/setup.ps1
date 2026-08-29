param(
    [string]$BinDir = "$HOME\.local\bin",
    [switch]$WithPdfTools
)

$ErrorActionPreference = "Stop"

$SourceDir = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$BinaryName = "agents-infra.exe"
$ModelHarnessBinaryName = "model-harness.exe"
$BuildOutput = Join-Path $SourceDir ".temp\bin\$BinaryName"
$ModelHarnessBuildOutput = Join-Path $SourceDir ".temp\bin\$ModelHarnessBinaryName"
$ConfigDir = Join-Path ([Environment]::GetFolderPath("ApplicationData")) "agents-infra"
$InstallStatePath = Join-Path $ConfigDir "install.json"
$BuildVersion = "dev"
$BuildCommit = "unknown"
$BuildDate = [DateTime]::UtcNow.ToString("yyyy-MM-ddTHH:mm:ssZ")
$BuildLdflags = ""

function Write-Info([string]$Message) {
    Write-Host $Message -ForegroundColor Green
}

function Write-Warn([string]$Message) {
    Write-Host $Message -ForegroundColor Yellow
}

function Ensure-Go {
    if (Get-Command go -ErrorAction SilentlyContinue) {
        Write-Info "Go already installed: $(go version)"
        return
    }

    if (-not (Get-Command winget -ErrorAction SilentlyContinue)) {
        throw "Go is missing and winget is not available. Install Go first."
    }

    Write-Warn "Go not found. Installing via winget..."
    winget install --exact --id GoLang.Go --accept-package-agreements --accept-source-agreements
    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        throw "Go install completed but go is still not on PATH. Restart the shell and rerun setup."
    }
    Write-Info "Go installed: $(go version)"
}

function Get-VersionMetadata {
    if (Get-Command git -ErrorAction SilentlyContinue) {
        try {
            Push-Location $SourceDir
            try {
                $script:BuildVersion = (git describe --tags --always 2>$null)
                if (-not $script:BuildVersion) {
                    $script:BuildVersion = "dev"
                }

                $script:BuildCommit = (git rev-parse --short HEAD 2>$null)
                if (-not $script:BuildCommit) {
                    $script:BuildCommit = "unknown"
                }
            }
            finally {
                Pop-Location
            }
        }
        catch {
            $script:BuildVersion = "dev"
            $script:BuildCommit = "unknown"
        }
    }

    $script:BuildLdflags = "-X main.Version=$script:BuildVersion -X main.Commit=$script:BuildCommit -X main.BuildDate=$script:BuildDate"
}

function Build-Cli {
    Write-Info "Building $BinaryName ..."
    New-Item -ItemType Directory -Force -Path (Split-Path -Parent $BuildOutput) | Out-Null
    go -C (Join-Path $SourceDir "tools\agents-infra") build -trimpath -ldflags $BuildLdflags -o $BuildOutput .
    Write-Info "Built: $BuildOutput"

    Write-Info "Building $ModelHarnessBinaryName ..."
    go -C (Join-Path $SourceDir "tools\agents-infra") build -trimpath -ldflags $BuildLdflags -o $ModelHarnessBuildOutput .\cmd\model-harness
    Write-Info "Built: $ModelHarnessBuildOutput"
}

function Install-Binary {
    New-Item -ItemType Directory -Force -Path $BinDir | Out-Null
    Copy-Item $BuildOutput (Join-Path $BinDir $BinaryName) -Force
    Write-Info "Installed binary: $(Join-Path $BinDir $BinaryName)"
    Copy-Item $ModelHarnessBuildOutput (Join-Path $BinDir $ModelHarnessBinaryName) -Force
    Write-Info "Installed binary: $(Join-Path $BinDir $ModelHarnessBinaryName)"
}

function Write-InstallState {
    New-Item -ItemType Directory -Force -Path $ConfigDir | Out-Null
    $Payload = @{
        repoPath = $SourceDir
        binDir = $BinDir
        platform = "windows"
        arch = [System.Runtime.InteropServices.RuntimeInformation]::ProcessArchitecture.ToString().ToLowerInvariant()
        version = $BuildVersion
        commit = $BuildCommit
        buildDate = $BuildDate
    } | ConvertTo-Json
    Set-Content -Path $InstallStatePath -Value $Payload
    Write-Info "Install state: $InstallStatePath"
}

function Ensure-UserPath {
    $CurrentUserPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $Parts = @()
    if ($CurrentUserPath) {
        $Parts = $CurrentUserPath -split ';'
    }

    if ($Parts -notcontains $BinDir) {
        $NewPath = (($Parts + $BinDir) | Where-Object { $_ -and $_.Trim() -ne "" } | Select-Object -Unique) -join ';'
        [Environment]::SetEnvironmentVariable("Path", $NewPath, "User")
        Write-Warn "Added $BinDir to the user PATH. Restart the shell if needed."
    }

    if (($env:Path -split ';') -notcontains $BinDir) {
        $env:Path = "$BinDir;$env:Path"
    }
}

function Verify-Install {
    $InstalledBinary = Join-Path $BinDir $BinaryName
    $InstalledModelHarness = Join-Path $BinDir $ModelHarnessBinaryName
    if (-not (Test-Path $InstalledBinary)) {
        throw "Missing installed binary: $InstalledBinary"
    }
    if (-not (Test-Path $InstalledModelHarness)) {
        throw "Missing installed binary: $InstalledModelHarness"
    }

    & $InstalledBinary version | Out-Null
    & $InstalledModelHarness version | Out-Null
    & $InstalledBinary setup global --source-dir $SourceDir | Out-Null
    & $InstalledBinary doctor global | Out-Null
    Write-Info "Verified binary and global setup"
}

Write-Host ""
Write-Info "=== relux-agents-infra setup ==="
Write-Host ""
Ensure-Go
Get-VersionMetadata
Build-Cli
Install-Binary
Write-InstallState
if ($WithPdfTools) {
    Write-Warn "PDF toolchain bootstrap is not automated on Windows yet."
}
Ensure-UserPath
Verify-Install
Write-Host ""
Write-Info "=== Done ==="
