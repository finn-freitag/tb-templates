#!/usr/bin/env bash
# Build a Windows golden image for the Tossbox libvirt driver.
#
#   ./build.sh <windows-11-pro|windows-server-2025> <windows.iso> [virtio-win.iso]
#
# Needs: packer, qemu (KVM), swtpm, xorriso, OVMF/edk2 firmware.
# On NixOS:  NIXPKGS_ALLOW_UNFREE=1 nix-shell ../../shell.nix --arg withPacker true
set -euo pipefail
cd "$(dirname "$0")"

name=${1:?usage: $0 <windows-11-pro|windows-server-2025> <windows.iso> [virtio-win.iso]}
iso=${2:?missing Windows ISO}
virtio_iso=${3:-}
vars="$name.pkrvars.hcl"
[[ -f $vars ]] || { echo "unknown image $name (no $vars)" >&2; exit 1; }
[[ -f $iso ]] || { echo "ISO not found: $iso" >&2; exit 1; }

mkdir -p build
if [[ -z $virtio_iso ]]; then
  virtio_iso=build/virtio-win.iso
  if [[ ! -f $virtio_iso ]]; then
    echo "Downloading virtio-win (stable)..."
    curl -fL -o "$virtio_iso.part" https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/stable-virtio/virtio-win.iso
    mv "$virtio_iso.part" "$virtio_iso"
  fi
fi
if [[ ! -f build/virtio/virtio-win-guest-tools.exe ]]; then
  rm -rf build/virtio && mkdir -p build/virtio
  xorriso -osirrox on -indev "$virtio_iso" -extract / build/virtio >/dev/null
  chmod -R u+w build/virtio
fi

# Locate UEFI firmware (override with OVMF_CODE / OVMF_VARS).
find_fw() {
  for f in "$@"; do [[ -f $f ]] && { echo "$f"; return; }; done
}
qemu_share="$(dirname "$(readlink -f "$(command -v qemu-system-x86_64)")")/../share/qemu"
OVMF_CODE=${OVMF_CODE:-$(find_fw /usr/share/OVMF/OVMF_CODE_4M.fd /usr/share/OVMF/OVMF_CODE.fd /usr/share/edk2/ovmf/OVMF_CODE.fd /usr/share/edk2/x64/OVMF_CODE.4m.fd "$qemu_share/edk2-x86_64-code.fd")}
OVMF_VARS=${OVMF_VARS:-$(find_fw /usr/share/OVMF/OVMF_VARS_4M.fd /usr/share/OVMF/OVMF_VARS.fd /usr/share/edk2/ovmf/OVMF_VARS.fd /usr/share/edk2/x64/OVMF_VARS.4m.fd "$qemu_share/edk2-i386-vars.fd")}
[[ -n $OVMF_CODE && -n $OVMF_VARS ]] || { echo "OVMF firmware not found; set OVMF_CODE and OVMF_VARS" >&2; exit 1; }

packer init .
packer build \
  -var-file="$vars" \
  -var "iso_url=$(readlink -f "$iso")" \
  -var "virtio_dir=$(readlink -f build/virtio)" \
  -var "efi_firmware_code=$OVMF_CODE" \
  -var "efi_firmware_vars=$OVMF_VARS" \
  ${PACKER_EXTRA_ARGS:-} \
  .

echo
echo "Done: output/$name/$name.qcow2"
echo "Copy it to each libvirt node:  <data_dir>/images/$name.qcow2"
