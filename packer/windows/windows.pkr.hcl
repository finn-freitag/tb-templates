// Builds a sysprepped Windows golden image (qcow2) for the Tossbox libvirt driver.
//
// The image contains: VirtIO drivers + QEMU guest agent, OpenSSH server,
// RDP with NLA, and cloudbase-init configured for NoCloud (cidata) metadata.
// It is generalized with sysprep, so every device created from it runs the
// specialize/OOBE passes and cloudbase-init on first boot.
//
// Usage: see README.md (./build.sh windows-11-pro /path/to/Win11.iso)

packer {
  required_plugins {
    qemu = {
      source  = "github.com/hashicorp/qemu"
      version = ">= 1.1.8"
    }
  }
}

variable "image_name" {
  type        = string
  description = "Output file name without extension; must match the local: file in the template (windows-11-pro, windows-server-2025)."
}

variable "windows_image" {
  type        = string
  description = "Name of the image inside install.wim, e.g. \"Windows 11 Pro\" or \"Windows Server 2025 Standard (Desktop Experience)\"."
}

variable "product_key" {
  type        = string
  default     = ""
  description = "Key used only to select the edition during setup (Microsoft's generic install / KMS client keys). The real license is applied per device by Tossbox."
}

variable "iso_url" {
  type        = string
  description = "Path or URL of the Windows installation ISO."
}

variable "iso_checksum" {
  type        = string
  default     = "none"
  description = "Checksum of the ISO (e.g. sha256:...), or \"none\"."
}

variable "virtio_dir" {
  type        = string
  description = "Directory containing the extracted virtio-win ISO (build.sh extracts it)."
}

variable "build_password" {
  type        = string
  sensitive   = true
  default     = "TossboxBuild-Only-1!"
  description = "Password of the temporary \"packer\" build account (WinRM). The account is deleted on first boot of every device."
}

variable "disk_size" {
  type    = string
  default = "64G"
}

variable "memory" {
  type    = number
  default = 8192
}

variable "cpus" {
  type    = number
  default = 4
}

variable "efi_firmware_code" {
  type    = string
  default = "/usr/share/OVMF/OVMF_CODE_4M.fd"
}

variable "efi_firmware_vars" {
  type    = string
  default = "/usr/share/OVMF/OVMF_VARS_4M.fd"
}

variable "headless" {
  type    = bool
  default = true
}

variable "output_dir" {
  type    = string
  default = "output"
}

source "qemu" "windows" {
  iso_url      = var.iso_url
  iso_checksum = var.iso_checksum

  vm_name          = "${var.image_name}.qcow2"
  output_directory = "${var.output_dir}/${var.image_name}"
  format           = "qcow2"
  disk_size        = var.disk_size
  disk_interface   = "virtio"
  disk_compression = true
  disk_discard     = "unmap"
  net_device       = "virtio-net"
  machine_type     = "q35"
  accelerator      = "kvm"
  cpu_model        = "host"
  memory           = var.memory
  cpus             = var.cpus
  headless         = var.headless

  // UEFI + TPM 2.0 (swtpm) so Windows 11 installs without hardware-check bypasses.
  efi_boot          = true
  efi_firmware_code = var.efi_firmware_code
  efi_firmware_vars = var.efi_firmware_vars
  vtpm              = true

  // Second CD: autounattend.xml, build scripts and the VirtIO drivers.
  cd_label = "TBSETUP"
  cd_files = [
    "${path.root}/scripts/",
    "${var.virtio_dir}/",
  ]
  cd_content = {
    "autounattend.xml" = templatefile("${path.root}/autounattend.xml.pkrtpl", {
      windows_image  = var.windows_image
      product_key    = var.product_key
      build_password = var.build_password
    })
  }

  // UEFI shows "Press any key to boot from CD or DVD".
  boot_wait    = "3s"
  boot_command = ["<enter><wait><enter><wait><enter>"]

  communicator   = "winrm"
  winrm_username = "packer"
  winrm_password = var.build_password
  winrm_timeout  = "4h"
  winrm_use_ssl  = false

  // Generalize with cloudbase-init's unattend file and power off.
  shutdown_command = "powershell -NoProfile -ExecutionPolicy Bypass -File C:\\tossbox-build\\sysprep.ps1"
  shutdown_timeout = "1h"
}

build {
  sources = ["source.qemu.windows"]

  provisioner "powershell" {
    inline = [
      "New-Item -ItemType Directory -Force -Path C:\\tossbox-build | Out-Null",
      "$cd = (Get-Volume | Where-Object FileSystemLabel -eq 'TBSETUP').DriveLetter",
      "Copy-Item \"$${cd}:\\*.ps1\" C:\\tossbox-build\\ -Force",
      "Copy-Item \"$${cd}:\\*.conf\" C:\\tossbox-build\\ -Force",
    ]
  }

  provisioner "powershell" {
    scripts = [
      "${path.root}/scripts/install-virtio.ps1",
      "${path.root}/scripts/enable-rdp.ps1",
      "${path.root}/scripts/install-openssh.ps1",
    ]
  }

  provisioner "windows-restart" {
    restart_timeout = "30m"
  }

  provisioner "powershell" {
    scripts = [
      "${path.root}/scripts/install-cloudbase-init.ps1",
      "${path.root}/scripts/cleanup.ps1",
    ]
  }
}
