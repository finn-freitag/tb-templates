# Packer is unfree (BSL) in nixpkgs. To get it (plus QEMU/swtpm for image builds) run:
#   NIXPKGS_ALLOW_UNFREE=1 nix-shell --arg withPacker true
{ pkgs ? import <nixpkgs> {}, withPacker ? false }:
pkgs.mkShell {
  packages = [ pkgs.go ]
    ++ pkgs.lib.optionals withPacker [ pkgs.packer pkgs.qemu pkgs.swtpm pkgs.xorriso ];
  # This module is standalone and not part of the parent go.work.
  GOWORK = "off";
}
