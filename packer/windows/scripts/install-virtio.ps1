# VirtIO drivers (balloon, serial, gpu, ...) and the QEMU guest agent from the
# virtio-win guest tools installer on the TBSETUP CD.
$ErrorActionPreference = 'Stop'
$cd = (Get-Volume | Where-Object FileSystemLabel -eq 'TBSETUP').DriveLetter
$tools = "$($cd):\virtio-win-guest-tools.exe"
if (-not (Test-Path $tools)) { throw "virtio-win-guest-tools.exe not found on TBSETUP CD" }
Start-Process -FilePath $tools -ArgumentList '/install', '/quiet', '/norestart' -Wait
Get-Service QEMU-GA -ErrorAction SilentlyContinue | Set-Service -StartupType Automatic
