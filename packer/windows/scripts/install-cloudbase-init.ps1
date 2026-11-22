# cloudbase-init configured for the NoCloud "cidata" ISO that the Tossbox agent
# attaches to every device (meta-data: hostname + SSH keys, user-data:
# #ps1_sysnative script that sets the password, product key and activation).
$ErrorActionPreference = 'Stop'
$msi = 'C:\tossbox-build\CloudbaseInitSetup_x64.msi'
$url = 'https://cloudbase.it/downloads/CloudbaseInitSetup_Stable_x64.msi'
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
Invoke-WebRequest -Uri $url -OutFile $msi -UseBasicParsing
$msiArgs = @('/i', $msi, '/qn', '/norestart', '/l*v', 'C:\tossbox-build\cloudbase-init-install.log',
          'RUN_SERVICE_AS_LOCAL_SYSTEM=1', 'LOGGINGSERIALPORTNAME=COM1', 'USERNAME=tossbox', 'USERGROUPS=Administrators')
$p = Start-Process msiexec.exe -ArgumentList $msiArgs -Wait -PassThru
if ($p.ExitCode -ne 0) { throw "cloudbase-init install failed: $($p.ExitCode)" }

$conf = 'C:\Program Files\Cloudbase Solutions\Cloudbase-Init\conf'
Copy-Item 'C:\tossbox-build\cloudbase-init.conf' "$conf\cloudbase-init.conf" -Force
Copy-Item 'C:\tossbox-build\cloudbase-init-unattend.conf' "$conf\cloudbase-init-unattend.conf" -Force
