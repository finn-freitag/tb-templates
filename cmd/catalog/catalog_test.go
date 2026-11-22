package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const validLinux = `id: debian-13
name: Debian 13
version: "1"
family: linux
guest_profile: cloud-init
readiness: ssh
default_user: tossbox
supported_services: [ssh, vnc, http, https]
min_resources: { vcpus: 1, ram_mb: 512, storage_gb: 8 }
default_resources: { vcpus: 2, ram_mb: 2048, storage_gb: 20 }
license: false
arch: [x86_64]
images:
  libvirt:
    url: https://example.org/debian.qcow2
    format: qcow2
params:
  os_variant: debian13
`

func TestParseValid(t *testing.T) {
	tpl, err := parse([]byte(validLinux))
	if err != nil {
		t.Fatal(err)
	}
	if tpl.ID != "debian-13" || tpl.Images["libvirt"].Format != "qcow2" {
		t.Fatalf("unexpected %+v", tpl)
	}
}

func TestParseInvalid(t *testing.T) {
	cases := map[string]struct{ from, to, want string }{
		"bad id":          {"id: debian-13", "id: Debian_13", "id"},
		"unknown field":   {"license: false", "license: false\nfoo: bar", "foo"},
		"bad family":      {"family: linux", "family: beos", "family"},
		"profile family":  {"guest_profile: cloud-init", "guest_profile: cloudbase-init", "not valid for family"},
		"unknown service": {"[ssh, vnc, http, https]", "[ssh, telnet]", "unknown service"},
		"adb on linux":    {"[ssh, vnc, http, https]", "[ssh, adb]", "adb"},
		"readiness svc":   {"readiness: ssh", "readiness: rdp", "requires that service"},
		"min > default":   {"vcpus: 1, ram_mb: 512", "vcpus: 4, ram_mb: 512", "exceed"},
		"unknown driver":  {"libvirt:", "vmware:", "unknown driver"},
		"bad scheme":      {"https://example.org/debian.qcow2", "ftp://example.org/x", "scheme"},
		"bad format":      {"format: qcow2", "format: vmdk", "format"},
		"bad sha":         {"format: qcow2", "format: qcow2\n    sha256: abc", "sha256"},
		"license linux":   {"license: false", "license: true", "license"},
		"bad arch":        {"arch: [x86_64]", "arch: [mips]", "arch"},
		"family driver":   {"libvirt:\n    url: https://example.org/debian.qcow2", "orchard:\n    url: oci://ghcr.io/x/y", "cannot run family"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			src := strings.Replace(validLinux, c.from, c.to, 1)
			if src == validLinux {
				t.Fatalf("replacement %q not applied", c.from)
			}
			_, err := parse([]byte(src))
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("want error containing %q, got %v", c.want, err)
			}
		})
	}
}

func TestPhoneTemplateNoResources(t *testing.T) {
	src := `id: iphone
name: iPhone
version: "1"
family: ios
guest_profile: phone
readiness: none
supported_services: []
min_resources: { vcpus: 0, ram_mb: 0, storage_gb: 0 }
default_resources: { vcpus: 0, ram_mb: 0, storage_gb: 0 }
license: false
arch: [x86_64, arm64]
images:
  usb-phone:
    url: ""
    format: none
`
	if _, err := parse([]byte(src)); err != nil {
		t.Fatal(err)
	}
}

func TestRunWriteAndCheck(t *testing.T) {
	dir := t.TempDir()
	tdir := filepath.Join(dir, "templates")
	if err := os.Mkdir(tdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tdir, "debian-13.yaml"), []byte(validLinux), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "catalog.json")
	now := time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC)

	if err := run(tdir, out, true, now); err == nil {
		t.Fatal("check must fail without catalog")
	}
	if err := run(tdir, out, false, now); err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(out)
	if err := run(tdir, out, true, now); err != nil {
		t.Fatal(err)
	}
	// Re-running must not change generated_at when nothing changed.
	if err := run(tdir, out, false, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(out)
	if string(first) != string(second) {
		t.Fatal("catalog rewritten although templates are unchanged")
	}
	// Changing a template makes check fail.
	changed := strings.Replace(validLinux, `version: "1"`, `version: "2"`, 1)
	if err := os.WriteFile(filepath.Join(tdir, "debian-13.yaml"), []byte(changed), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run(tdir, out, true, now); err == nil {
		t.Fatal("check must fail after change")
	}
	// File name must match id.
	if err := os.Rename(filepath.Join(tdir, "debian-13.yaml"), filepath.Join(tdir, "other.yaml")); err != nil {
		t.Fatal(err)
	}
	if err := run(tdir, out, true, now); err == nil || !strings.Contains(err.Error(), "file name") {
		t.Fatalf("want file name error, got %v", err)
	}
}
