# Packer shutdown_command: generalize with cloudbase-init's unattend file and power off.
$unattend = 'C:\Program Files\Cloudbase Solutions\Cloudbase-Init\conf\Unattend.xml'
& "$env:SystemRoot\System32\Sysprep\sysprep.exe" /generalize /oobe /shutdown /quiet "/unattend:$unattend"
