# Render mockup HTML ke JPG memakai Chrome headless (Mendukung Dark & Light Theme).
$ErrorActionPreference = "Continue"
Add-Type -AssemblyName System.Drawing

$chrome = "C:\Program Files\Google\Chrome\Application\chrome.exe"
if (-not (Test-Path $chrome)) {
    Write-Error "Chrome executable tidak ditemukan di $chrome"
    exit 1
}

$src = Join-Path $PSScriptRoot "src"
$out = $PSScriptRoot
$tmp = Join-Path $env:TEMP "sw-mock"
New-Item -ItemType Directory -Force $tmp | Out-Null

$codec = [System.Drawing.Imaging.ImageCodecInfo]::GetImageEncoders() | Where-Object MimeType -eq "image/jpeg"
$params = New-Object System.Drawing.Imaging.EncoderParameters 1
$params.Param[0] = New-Object System.Drawing.Imaging.EncoderParameter ([System.Drawing.Imaging.Encoder]::Quality, [long]94)

# Copy static assets (css, js) ke tmp agar relative path iframe & script tetap valid
Copy-Item (Join-Path $src "*.css") $tmp -Force
Copy-Item (Join-Path $src "*.js") $tmp -Force

$files = Get-ChildItem $src -Filter *.html | Where-Object { $args.Count -eq 0 -or $args -contains $_.BaseName }

foreach ($file in $files) {
    $raw = Get-Content $file.FullName -Raw
    
    # 1. Render Dark Theme (Default)
    $darkContent = $raw -replace '<html([^>]*)>', '<html$1 data-theme="dark">'
    $darkTmp = Join-Path $tmp ($file.BaseName + "-dark.html")
    Set-Content -Path $darkTmp -Value $darkContent -Encoding UTF8
    
    $darkPng = Join-Path $tmp ($file.BaseName + "-dark.png")
    $darkUrl = "file:///" + ($darkTmp -replace "\\", "/")
    
    & $chrome --headless=new --disable-gpu --hide-scrollbars --force-device-scale-factor=1 `
        --window-size=1440,900 --virtual-time-budget=2000 "--screenshot=$darkPng" $darkUrl 2>$null | Out-Null
    
    if (Test-Path $darkPng) {
        $img = [System.Drawing.Image]::FromFile($darkPng)
        $jpgDefault = Join-Path $out ($file.BaseName + ".jpg")
        $img.Save($jpgDefault, $codec, $params)
        $img.Dispose()
        Write-Output "OK (Dark): $jpgDefault"
    }

    # 2. Render Light Theme
    $lightContent = $raw -replace '<html([^>]*)>', '<html$1 data-theme="light">'
    $lightTmp = Join-Path $tmp ($file.BaseName + "-light.html")
    Set-Content -Path $lightTmp -Value $lightContent -Encoding UTF8
    
    $lightPng = Join-Path $tmp ($file.BaseName + "-light.png")
    $lightUrl = "file:///" + ($lightTmp -replace "\\", "/")
    
    & $chrome --headless=new --disable-gpu --hide-scrollbars --force-device-scale-factor=1 `
        --window-size=1440,900 --virtual-time-budget=2000 "--screenshot=$lightPng" $lightUrl 2>$null | Out-Null
    
    if (Test-Path $lightPng) {
        $img = [System.Drawing.Image]::FromFile($lightPng)
        $jpgLight = Join-Path $out ($file.BaseName + "-light.jpg")
        $img.Save($jpgLight, $codec, $params)
        $img.Dispose()
        Write-Output "OK (Light): $jpgLight"
    }
}

Write-Output "Semua capture berhasil di-render ke $out!"
