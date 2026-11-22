# Windows golden images

The `libvirt` Windows templates (`windows-11-pro`, `windows-server-2025`) use `local:` images: a sysprepped qcow2 that you build once with Packer and copy to every libvirt node. Windows images can't be distributed publicly because of licensing, so they aren't downloaded automatically.

The `docker-qemu` variants of the same templates don't need this. They use [dockur/windows](https://github.com/dockur/windows), which installs Windows from Microsoft's ISO on first start (slow, 10–20 min per device).

## What's in the image

| Component | Purpose |
|---|---|
| VirtIO drivers + QEMU guest agent | Disk/network performance, clean shutdown, IP reporting |
| OpenSSH server (PowerShell as default shell) | `ssh` service |
| Remote Desktop with NLA | `rdp` service |
| **cloudbase-init** (NoCloud metadata) | Per-device provisioning from the agent's `cidata` ISO: hostname, user `tossbox` (Administrators), SSH keys, password + product key via user-data |
| Sysprep (generalize + OOBE) | Every device gets a unique SID and runs setup on first boot |

The temporary build account `packer` and the build's WinRM configuration are removed on each device's first boot (`C:\Windows\Setup\Scripts\SetupComplete.cmd`).

## Build

Requirements: Linux host with KVM, `packer`, `qemu`, `swtpm`, `xorriso`, OVMF/edk2 firmware, about 80 GB free disk space, and internet access (the virtio-win ISO, the cloudbase-init MSI and the Windows 11 OpenSSH Feature on Demand are downloaded during the build).

On NixOS (Packer is unfree in nixpkgs):

```bash
NIXPKGS_ALLOW_UNFREE=1 nix-shell ../../shell.nix --arg withPacker true
```

Get the ISO from Microsoft (Windows 11 multi-edition ISO, or the Windows Server 2025 evaluation ISO), then:

```bash
./build.sh windows-11-pro ~/Downloads/Win11_25H2_English_x64.iso
```

```bash
./build.sh windows-server-2025 ~/Downloads/WindowsServer2025_x64_EN_Eval.iso
```

Optional third argument: a local `virtio-win.iso` (otherwise the current stable one is downloaded into `build/`). The firmware is detected automatically; override with `OVMF_CODE=... OVMF_VARS=...`. Extra Packer flags: `PACKER_EXTRA_ARGS="-var headless=false"` to watch the install over VNC.

Result: `output/<name>/<name>.qcow2`.

## Install on a node

Copy the image into the agent's image directory on **every libvirt node** that should offer the template:

```bash
scp output/windows-11-pro/windows-11-pro.qcow2 homelab:/var/lib/tb-agent/images/windows-11-pro.qcow2
```

(`<data_dir>/images/`, where `data_dir` comes from the node config; default `/var/lib/tb-agent`.) The file name must match the `local:` URL in the template. Devices are created as linked clones of this file, so never modify it while devices exist. Build a new file and bump the template `version` instead.

## Licensing

- The keys in the `*.pkrvars.hcl` files are Microsoft's **public generic installation / KMS client keys**. They only select the edition during setup and don't activate Windows.
- Each device gets its license from Tossbox: an own key, a key from a node license pool, or none (unactivated). The agent passes the key in the cloudbase-init user-data script, which runs `slmgr /ipk` + `/ato`.
- You're responsible for having valid licenses for the Windows installations you run. Evaluation ISOs expire (Server: 180 days).

## Files

| File | Purpose |
|---|---|
| `windows.pkr.hcl` | Packer template (QEMU builder, UEFI + vTPM, WinRM) |
| `autounattend.xml.pkrtpl` | Install-time answer file (disk layout, VirtIO drivers, build account) |
| `*.pkrvars.hcl` | Per-edition variables (`image_name`, `windows_image`, setup key) |
| `scripts/*.ps1` | Build steps: WinRM, VirtIO, RDP, OpenSSH, cloudbase-init, cleanup, sysprep |
| `scripts/cloudbase-init*.conf` | cloudbase-init configuration (NoCloud metadata service, plugins) |
| `build.sh` | Wrapper: downloads/extracts virtio-win, finds firmware, runs Packer |
