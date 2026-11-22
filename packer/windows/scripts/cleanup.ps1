# Final cleanup before sysprep. WinRM keeps running for the current Packer
# session but will not start on devices.
$ErrorActionPreference = 'Continue'

# Remove the temporary build account and its autologon on each device's first boot.
$scripts = 'C:\Windows\Setup\Scripts'
New-Item -ItemType Directory -Force -Path $scripts | Out-Null
@'
@echo off
net user packer /delete
reg delete "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon" /v DefaultPassword /f
reg add "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon" /v AutoAdminLogon /t REG_SZ /d 0 /f
reg delete "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System" /v LocalAccountTokenFilterPolicy /f
rmdir /s /q C:\Users\packer
rmdir /s /q C:\tossbox-build
'@ | Set-Content -Path "$scripts\SetupComplete.cmd" -Encoding ASCII

Set-Service WinRM -StartupType Disabled
Get-NetFirewallRule -DisplayName 'WinRM (Packer build)' -ErrorAction SilentlyContinue | Remove-NetFirewallRule
winrm set winrm/config/service '@{AllowUnencrypted="false"}' | Out-Null
winrm set winrm/config/service/auth '@{Basic="false"}' | Out-Null

# Shrink the image.
Remove-Item -Recurse -Force "$env:TEMP\*" -ErrorAction SilentlyContinue
Remove-Item -Force 'C:\tossbox-build\*.msi' -ErrorAction SilentlyContinue
Dism.exe /Online /Cleanup-Image /StartComponentCleanup /ResetBase | Out-Null
Optimize-Volume -DriveLetter C -ReTrim -ErrorAction SilentlyContinue
