# Linux images

Linux templates use the **official cloud images** of each distribution directly (`https://` URLs in `templates/*.yaml`). The agent downloads and caches them, creates every device as a linked clone, and provisions it with cloud-init from a NoCloud `cidata` seed (hostname, user `tossbox` with password, SSH keys). No build step is needed.

## Customizing

If you want a Linux image with extra software preinstalled (for example a CI image with a toolchain):

1. Build a qcow2 from the official cloud image, e.g. with Packer's QEMU builder (`disk_image = true`, `iso_url` = the cloud image) and shell provisioners, or with `virt-customize --install ...`.
2. Make sure cloud-init stays installed, and reset it at the end (`cloud-init clean --logs --machine-id`) so it runs again on each device.
3. Copy the result to `<data_dir>/images/<name>.qcow2` on your libvirt nodes.
4. Add a template whose `images.libvirt.url` is `local:<name>.qcow2`, or host the file on HTTPS and use that URL (optionally with `sha256`).
