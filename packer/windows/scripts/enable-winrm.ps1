# Runs at first logon of the build account (from autounattend.xml).
# Enables unencrypted WinRM for Packer only; cleanup.ps1 disables WinRM again.
$ErrorActionPreference = 'Stop'
Get-NetConnectionProfile | Set-NetConnectionProfile -NetworkCategory Private
# Local (non built-in) admins need this to get an elevated token over WinRM.
New-ItemProperty -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System' -Name LocalAccountTokenFilterPolicy -Value 1 -PropertyType DWord -Force | Out-Null
winrm quickconfig -q
winrm set winrm/config/service '@{AllowUnencrypted="true"}'
winrm set winrm/config/service/auth '@{Basic="true"}'
winrm set winrm/config/winrs '@{MaxMemoryPerShellMB="2048"}'
New-NetFirewallRule -DisplayName 'WinRM (Packer build)' -Direction Inbound -Protocol TCP -LocalPort 5985 -Action Allow | Out-Null
Set-Service WinRM -StartupType Automatic
Restart-Service WinRM
