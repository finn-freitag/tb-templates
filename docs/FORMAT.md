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
family: linux                 # linux | windows | android | macos | ios
icon: debian                  # optional, UI hint
guest_profile: cloud-init     # cloud-init | cloudbase-init | android | tart | phone
readiness: ssh                # ssh | rdp | adb | none
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
- Validation rules (enforced by `cmd/catalog`): RDP/VNC only for `linux`, `windows` and `macos`; the readiness service must be in `supported_services`; each driver only runs certain families (libvirt/docker-qemu: linux, windows; lxc: linux; android: android; orchard: macos; usb-phone: android, ios).

## Known params

| Param | Used by | Meaning |
|---|---|---|
| `os_variant` | libvirt | osinfo hint |
| `docker_qemu_boot` | docker-qemu | `BOOT` value for qemux/qemu (cloud image URL) |
| `dockur_version` | docker-qemu | `VERSION` value for dockurr/windows (`11`, `2025`, ...) |
| `tpm`, `secure_boot` | libvirt (Windows) | `"true"`/`"false"` |
| `backend` | android | `redroid` or `cuttlefish` |
| `cvd_branch`, `cvd_target` | android (cuttlefish) | build to fetch |
| `initial_password` | orchard | default password of the image, changed by the agent after boot |

## Image URL schemes

| Scheme | Meaning |
|---|---|
| `https://…` | downloaded and cached by the agent (checksum verified if `sha256` set) |
| `local:<file>` | file that must exist in the agent's image directory, e.g. a licensed Windows golden image built with Packer from `packer/` |
| `docker://<ref>` | container image (docker-qemu, android/redroid) |
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
