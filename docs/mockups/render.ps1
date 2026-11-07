# Render mockup HTML ke JPG memakai Chrome headless.
$ErrorActionPreference = "Continue"  # Chrome menulis log ke stderr
Add-Type -AssemblyName System.Drawing
$chrome = "C:\Program Files\Google\Chrome\Application\chrome.exe"
$src = Join-Path $PSScriptRoot "src"
$out = $PSScriptRoot
$tmp = Join-Path $env:TEMP "sw-mock"
New-Item -ItemType Directory -Force $tmp | Out-Null

$codec = [System.Drawing.Imaging.ImageCodecInfo]::GetImageEncoders() | Where-Object MimeType -eq "image/jpeg"
$params = New-Object System.Drawing.Imaging.EncoderParameters 1
$params.Param[0] = New-Object System.Drawing.Imaging.EncoderParameter ([System.Drawing.Imaging.Encoder]::Quality, [long]92)

Get-ChildItem $src -Filter *.html | Where-Object { $args.Count -eq 0 -or $args -contains $_.BaseName } | ForEach-Object {
    $png = Join-Path $tmp ($_.BaseName + ".png")
    $url = "file:///" + ($_.FullName -replace "\\", "/")
    & $chrome --headless=new --disable-gpu --hide-scrollbars --force-device-scale-factor=1 `
        --window-size=1440,900 --virtual-time-budget=2000 "--screenshot=$png" $url 2>$null | Out-Null
    $img = [System.Drawing.Image]::FromFile($png)
    $jpg = Join-Path $out ($_.BaseName + ".jpg")
    $img.Save($jpg, $codec, $params)
    $img.Dispose()
    Write-Output "OK $jpg"
}
