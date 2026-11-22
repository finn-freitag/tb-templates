# Template format

A template ("egg") describes **what** runs on a device: the OS image, how the guest is provisioned, and which drivers and services it supports.

Sources:
- Each template is one YAML file in `templates/<id>.yaml`.
- `catalog.json` at the repo root is generated from all template files (`go run ./cmd/catalog`, checked in CI). The dashboard loads catalogs from `TOSSBOX_TEMPLATE_SOURCES`, which can be URLs to `catalog.json` or local directories with `*.yaml` files. The default is `https://raw.githubusercontent.com/finn-freitag/tb-templates/main/catalog.json`.

## Fields

```yaml
id: debian-13                 # unique, [a-z0-9-]
name: Debian 13 (Trixie)
description: Official Debian generic cloud image
version: "1"                  # bump when the image or provisioning changes
family: linux                 # linux | windows | android | macos | ios | chromeos | browser
icon: debian                  # optional, UI hint
guest_profile: cloud-init     # cloud-init | cloudbase-init | android | tart | phone | none
readiness: ssh                # ssh | rdp | adb | vnc | http | https | none (vnc/http/https: TCP connect)
default_user: tossbox         # user created / used for ssh, rdp, vnc
supported_services: [ssh, vnc, http, https]
service_ports:                # optional overrides of internal ports
  ssh: 22
min_resources:    { vcpus: 1, ram_mb: 512,  storage_gb: 8 }
default_resources: { vcpus: 2, ram_mb: 2048, storage_gb: 20 }
license: false                # true: Windows license options are offered
arch: [x86_64]                # host architectures the images run on
images:                       # one entry per compatible driver
  libvirt:
    url: https://cloud.debian.org/images/cloud/trixie/latest/debian-13-genericcloud-amd64.qcow2
    format: qcow2
    sha256: ""                # optional
  docker-qemu:
    url: docker://qemux/qemu:latest
    format: container
params:                       # free-form, passed to the driver (TemplateSpec.params)
  os_variant: debian13
```

- The list of compatible drivers is the set of keys in `images`.
- `params` are passed through as strings.

## Formats and special cases

- `format`: `qcow2`, `raw`, `img`, `iso`, `container` (docker images and lxc/incus images), `oci` (orchard/tart), `none` (usb-phone).
- `usb-phone` templates use `url: ""` and `format: none`; templates with the `phone` profile may have empty `supported_services` and zero resources.
- `https://` images ending in `.zip`, `.tar.xz`, `.tar.gz`, `.xz` or `.gz` are unpacked by the agent (zip/tar: the largest file is the disk image; `sha256` refers to the archive).
- Validation rules (enforced by `cmd/catalog`): RDP/VNC only for `linux`, `windows`, `macos`, `chromeos` and `browser`; the readiness service must be in `supported_services`; each driver only runs certain families (libvirt: linux, windows, chromeos; docker-qemu: linux, windows; lxc: linux; android: android; orchard: macos; usb-phone: android, ios; browser: browser).
- Guest profile `none`: the image is used as it is, without a seed (ChromeOS, browsers).

## Known params

Params are passed to the driver as `TemplateSpec.params`. Driver-specific keys are namespaced by driver; unknown keys are ignored.

| Param | Used by | Meaning |
|---|---|---|
| `os_variant` | libvirt | osinfo hint |
| `cloudinit.packages` | libvirt, docker-qemu, lxc (cloud-init profile) | extra packages installed on first boot (space or comma separated) |
| `cloudinit.script` | libvirt, docker-qemu, lxc (cloud-init profile) | shell script run once as root after the packages; output in `/var/log/tossbox-setup.log`. Combine with `readiness: rdp` (or another service the script starts) so the device only turns ready when the script is done |
| `tpm`, `secure_boot` | libvirt (Windows) | `"true"`/`"false"` |
| `disk_bus`, `video`, `net_model` | libvirt | disk bus (`virtio`, `sata`), video model, NIC model (`virtio`, `e1000e`) overrides |
| `firmware` | libvirt (non-Windows) | `efi`: boot with UEFI (secure boot off); default BIOS |
| `gpu` | dashboard | `required`: devices always get a GPU; `recommended`: GPU by default when a node offers one |
| `browser.env.<NAME>` | browser | container environment; `{password}`, `{user}` and `{hostname}` are replaced |
| `browser.shm_mb` | browser | `/dev/shm` size in MiB (Chromium needs ≥ 1024) |
| `docker_qemu.boot_image` | docker-qemu (Linux) | cloud image URL (qcow2/img), fetched by the agent, booted by qemux/qemu as `/boot.<ext>` together with the cidata seed |
| `docker_qemu.boot_image_sha256` | docker-qemu | optional checksum of the boot image |
| `docker_qemu.boot` | docker-qemu | qemux built-in OS name (no provisioning), only used without `boot_image` |
| `docker_qemu.version` | docker-qemu (Windows) | dockur `VERSION` (`11`, `2025`, …) |
| `docker_qemu.edition`, `docker_qemu.language` | docker-qemu (Windows) | dockur `EDITION`, `LANGUAGE` |
| `docker_qemu.prepared_storage` | docker-qemu (Windows) | `local:<dir>` with an installed dockur `/storage` snapshot (skips the install; keeps the snapshot's password) |
| `lxc.type` | lxc | `container` (default) or `virtual-machine` |
| `lxc.packages` | lxc | packages cloud-init installs (space or comma separated); default `openssh-server`, since `images:` cloud variants ship without an SSH server |
| `android.backend` | android | `redroid` or `cuttlefish` (a device param `backend` overrides it) |
| `redroid.width`, `redroid.height`, `redroid.dpi` | android (redroid) | display size / density |
| `redroid.requires_ashmem` | android (redroid) | `"true"`: the image needs the ashmem kernel driver; creation fails with a clear message on nodes without it |
| `orchard.image_username`, `orchard.image_password` | orchard | credentials baked into the Tart image; Orchard connects with them and the agent sets the generated password after boot |

Cuttlefish templates use `images.android.url: local:<dir>` (format `dir`): a directory below the node's `drivers.android.cuttlefish.images_dir` with images fetched via `cvd fetch` (see tb-agent `docs/drivers/android.md`).

## Image URL schemes

| Scheme | Meaning |
|---|---|
| `https://…` | downloaded and cached by the agent (checksum verified if `sha256` set) |
| `local:<file>` | file that must exist in the agent's image directory, e.g. a licensed Windows golden image built with Packer from `packer/` |
| `docker://<ref>` | container image (docker-qemu, android/redroid, browser) |
| `local:<dir>` (format `dir`) | android/cuttlefish: image directory below `images_dir` |
| `oci://<ref>` | OCI VM image (orchard/tart) |
| `incus:<remote>:<alias>` | Incus/LXC image, e.g. `incus:images:debian/13/cloud` (lxc driver) |

## Guest profiles

| Profile | Used for | How the agent provisions |
|---|---|---|
| `cloud-init` | Linux VMs and containers | NoCloud seed (ISO labeled `cidata`, or Incus `cloud-init.*` config): hostname, `default_user` with password, SSH keys, enable password SSH login |
| `cloudbase-init` | Windows | Same NoCloud `cidata` ISO, read by **cloudbase-init** that is preinstalled in the golden image. User data is a PowerShell script (`#ps1_sysnative`): set the password, product key + activation, enable RDP with NLA and OpenSSH |
| `android` | redroid / Cuttlefish | adb settings after boot |
| `tart` | macOS via Orchard | Orchard VM spec + SSH (password change via SSH after boot) |
| `phone` | physical phones | reset + setup flow of the usb-phone driver |

## Default internal service ports

`ssh` 22, `rdp` 3389, `vnc` 5900, `adb` 5555, `http` 80, `https` 443.

## catalog.json

```json
{
  "schema": 1,
  "generated_at": "2026-11-22T16:00:00Z",
  "templates": [ { "...same fields as YAML, as JSON..." } ]
}
```
