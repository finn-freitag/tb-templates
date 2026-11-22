# tb-templates

Device templates ("eggs") for [Tossbox](https://github.com/finn-freitag/tossbox), the self-hosted throwaway device service.

A template describes **what** runs on a device: the OS image per driver, how the guest is provisioned, which services (SSH, RDP, VNC, ADB, HTTP/HTTPS) it supports, and its resource minimums and defaults. **Where** a device runs is a node, and **how** it runs is a driver of the [tb-agent](https://github.com/finn-freitag/tb-agent).

- `templates/<id>.yaml`: one file per template, format in [docs/FORMAT.md](docs/FORMAT.md)
- `catalog.json`: all templates in one file, generated from the YAML files; this is what the dashboard loads
- `cmd/catalog`: validator and generator
- `packer/windows`: build the Windows golden images for the libvirt driver
- `packer/linux`: notes on custom Linux images (official cloud images are used directly)

## Templates

| ID | Name | Family | Drivers | Services |
|---|---|---|---|---|
| `almalinux-10` | AlmaLinux 10 | linux | `docker-qemu`, `libvirt` | ssh, vnc, http, https |
| `android-15-redroid` | Android 15 (redroid) | android | `android` | adb |
| `android-16-redroid` | Android 16 (redroid) | android | `android` | adb |
| `android-cuttlefish` | Android (Cuttlefish) | android | `android` | adb |
| `android-phone` | Android phone (physical) | android | `usb-phone` | adb |
| `archlinux` | Arch Linux | linux | `docker-qemu`, `libvirt` | ssh, vnc, http, https |
| `debian-12` | Debian 12 (Bookworm) | linux | `docker-qemu`, `libvirt` | ssh, vnc, http, https |
| `debian-13` | Debian 13 (Trixie) | linux | `docker-qemu`, `libvirt` | ssh, vnc, http, https |
| `fedora-45` | Fedora 45 Cloud | linux | `docker-qemu`, `libvirt` | ssh, vnc, http, https |
| `iphone` | iPhone (physical) | ios | `usb-phone` | – |
| `lxc-alpine-3-24` | Alpine 3.24 (container) | linux | `lxc` | ssh, http, https |
| `lxc-archlinux` | Arch Linux (container) | linux | `lxc` | ssh, http, https |
| `lxc-debian-13` | Debian 13 (container) | linux | `lxc` | ssh, http, https |
| `lxc-ubuntu-24-04` | Ubuntu 24.04 LTS (container) | linux | `lxc` | ssh, http, https |
| `macos-sequoia` | macOS Sequoia | macos | `orchard` | ssh, vnc |
| `macos-sequoia-xcode` | macOS Sequoia + Xcode | macos | `orchard` | ssh, vnc |
| `macos-tahoe` | macOS Tahoe | macos | `orchard` | ssh, vnc |
| `macos-tahoe-xcode` | macOS Tahoe + Xcode | macos | `orchard` | ssh, vnc |
| `rocky-10` | Rocky Linux 10 | linux | `docker-qemu`, `libvirt` | ssh, vnc, http, https |
| `ubuntu-24-04` | Ubuntu 24.04 LTS (Noble Numbat) | linux | `docker-qemu`, `libvirt` | ssh, vnc, http, https |
| `ubuntu-26-04` | Ubuntu 26.04 LTS (Resolute Raccoon) | linux | `docker-qemu`, `libvirt` | ssh, vnc, http, https |
| `windows-11-pro` | Windows 11 Pro | windows | `docker-qemu`, `libvirt` | rdp, ssh, vnc, http, https |
| `windows-server-2025` | Windows Server 2025 | windows | `docker-qemu`, `libvirt` | rdp, ssh, vnc, http, https |

Notes:
- **Windows (libvirt)** needs a golden image built with [packer/windows](packer/windows/README.md) and copied to each node. The docker-qemu variant installs Windows automatically through dockur/windows.
- **macOS** runs through Orchard on Apple Silicon Macs, max 2 VMs per Mac.
- **Physical phones** have no resources. The phone node leases a connected device.

## How the dashboard uses this

The dashboard reads template sources from `TOSSBOX_TEMPLATE_SOURCES` (comma separated). Each entry is either a URL to a `catalog.json` or a local directory containing `*.yaml` template files. The default is:

```
https://raw.githubusercontent.com/finn-freitag/tb-templates/main/catalog.json
```

To add your own templates without forking, mount a directory into the dashboard container and append it, e.g. `TOSSBOX_TEMPLATE_SOURCES=https://raw.githubusercontent.com/finn-freitag/tb-templates/main/catalog.json,/templates`. Admins can reload the sources from the dashboard (Templates → Reload).

## Adding a template

1. Create `templates/<id>.yaml` (the file name must equal `id`). Start from a similar template and read [docs/FORMAT.md](docs/FORMAT.md).
2. Use stable, official image URLs (`.../latest/...` style), and verify they exist.
3. Validate and regenerate the catalog:

   ```bash
   nix-shell --run 'go run ./cmd/catalog'
   ```

   (or `GOWORK=off go run ./cmd/catalog` with a local Go toolchain)
4. Commit both the YAML file and `catalog.json`. CI fails if `catalog.json` is out of date (`go run ./cmd/catalog -check`).

Bump `version` whenever the image or provisioning of an existing template changes.

## Validation rules (summary)

- Known families, guest profiles, readiness checks, services, drivers and architectures only; unknown YAML keys are rejected.
- The guest profile must fit the family (`cloudbase-init` only for Windows, `tart` for macOS, ...), and each driver must be able to run the family.
- Image URL schemes per driver: libvirt `https`/`local`, docker-qemu `docker`, lxc `incus`, android `docker`/`local`, orchard `oci`, usb-phone none.
- RDP/VNC only for desktop families, ADB only for Android; the readiness service must be supported.
- `min_resources` ≤ `default_resources`; `license: true` only for Windows.
