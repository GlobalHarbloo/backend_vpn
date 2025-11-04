# Generate platform app icons from assets/icon/app_icon.png
# Usage: run this script in PowerShell (Windows) from project root
# It will run `flutter pub get`, then `flutter_launcher_icons`, and attempt to create a Windows .ico
# if ImageMagick (magick) is available.

# Determine project root (script is in tools/)
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
$projectRoot = Resolve-Path (Join-Path $scriptDir '..')
Set-Location $projectRoot.Path

Write-Host "Project root: $($projectRoot.Path)"

# Backup existing icons (optional)
$timestamp = Get-Date -Format yyyyMMdd_HHmmss
$backupDir = Join-Path $projectRoot.Path "tools\icon_backups_$timestamp"
New-Item -ItemType Directory -Path $backupDir -Force | Out-Null
Write-Host "Created backup dir: $backupDir"

# List of Android mipmap paths to backup
$mipmaps = @(
    'android\app\src\main\res\mipmap-mdpi\ic_launcher.png',
    'android\app\src\main\res\mipmap-hdpi\ic_launcher.png',
    'android\app\src\main\res\mipmap-xhdpi\ic_launcher.png',
    'android\app\src\main\res\mipmap-xxhdpi\ic_launcher.png',
    'android\app\src\main\res\mipmap-xxxhdpi\ic_launcher.png'
)
foreach ($p in $mipmaps) {
    $full = Join-Path $projectRoot.Path $p
    if (Test-Path $full) {
        Copy-Item $full -Destination $backupDir -Force
        Write-Host "Backed up: $p"
    }
}
# Backup Windows ico
$winIco = Join-Path $projectRoot.Path 'windows\runner\resources\app_icon.ico'
if (Test-Path $winIco) {
    Copy-Item $winIco -Destination $backupDir -Force
    Write-Host "Backed up: $winIco"
}

# Backup iOS appiconset
$iosAppIcon = Join-Path $projectRoot.Path 'ios\Runner\Assets.xcassets\AppIcon.appiconset'
if (Test-Path $iosAppIcon) {
    Copy-Item $iosAppIcon -Destination $backupDir -Recurse -Force
    Write-Host "Backed up iOS AppIcon.appiconset"
}

# Run flutter pub get
Write-Host "Running flutter pub get..."
try {
    & flutter pub get
    if ($LASTEXITCODE -ne 0) { throw "flutter pub get failed (exit $LASTEXITCODE)" }
} catch {
    Write-Error "flutter pub get failed. Ensure Flutter is installed and on PATH. Error: $_"
    Exit 1
}

# Run flutter_launcher_icons
Write-Host "Running flutter_launcher_icons to generate platform icons..."
try {
    & flutter pub run flutter_launcher_icons:main
    if ($LASTEXITCODE -ne 0) { Write-Warning "flutter_launcher_icons exited with code $LASTEXITCODE — check pubspec.yaml configuration." } else { Write-Host "flutter_launcher_icons completed." }
} catch {
    Write-Warning "Failed to run flutter_launcher_icons: $_"
}

# Generate Windows .ico using ImageMagick if available
 $png = Join-Path $projectRoot.Path 'assets\icon\app_icon.png'
 $icoOut = Join-Path $projectRoot.Path 'windows\runner\resources\app_icon.ico'
 if (-Not (Test-Path $png)) {
    Write-Warning "Source PNG not found at $png. Place your icon there (recommended 1024x1024) and re-run the script."
 } else {
    $magick = Get-Command magick -ErrorAction SilentlyContinue
    if ($magick) {
        Write-Host "ImageMagick found — generating multi-size .ico..."
        try {
            & magick convert "$png" -define icon:auto-resize=256,128,64,48,32,16 "$icoOut"
            if (Test-Path $icoOut) { Write-Host "Generated Windows icon: $icoOut" } else { Write-Warning "Failed to generate Windows .ico" }
        } catch {
            Write-Warning "ImageMagick command failed: $_"
        }
    } else {
        Write-Warning "ImageMagick not found. Skipping .ico generation. Install ImageMagick or create .ico manually."
    }
 }

Write-Host "Icon generation step complete. Rebuild app to apply icons."