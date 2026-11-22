package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Resources are the resource fields of a template (see docs/FORMAT.md).
type Resources struct {
	VCPUs     int   `yaml:"vcpus" json:"vcpus"`
	RAMMB     int64 `yaml:"ram_mb" json:"ram_mb"`
	StorageGB int64 `yaml:"storage_gb" json:"storage_gb"`
}

// Image is the image entry for one driver.
type Image struct {
	URL    string `yaml:"url" json:"url"`
	Format string `yaml:"format" json:"format"`
	SHA256 string `yaml:"sha256,omitempty" json:"sha256,omitempty"`
}

// Template mirrors the YAML format in docs/FORMAT.md.
type Template struct {
	ID                string            `yaml:"id" json:"id"`
	Name              string            `yaml:"name" json:"name"`
	Description       string            `yaml:"description,omitempty" json:"description,omitempty"`
	Version           string            `yaml:"version" json:"version"`
	Family            string            `yaml:"family" json:"family"`
	Icon              string            `yaml:"icon,omitempty" json:"icon,omitempty"`
	GuestProfile      string            `yaml:"guest_profile" json:"guest_profile"`
	Readiness         string            `yaml:"readiness" json:"readiness"`
	DefaultUser       string            `yaml:"default_user,omitempty" json:"default_user,omitempty"`
	SupportedServices []string          `yaml:"supported_services" json:"supported_services"`
	ServicePorts      map[string]int    `yaml:"service_ports,omitempty" json:"service_ports,omitempty"`
	MinResources      Resources         `yaml:"min_resources" json:"min_resources"`
	DefaultResources  Resources         `yaml:"default_resources" json:"default_resources"`
	License           bool              `yaml:"license" json:"license"`
	Arch              []string          `yaml:"arch" json:"arch"`
	Images            map[string]Image  `yaml:"images" json:"images"`
	Params            map[string]string `yaml:"params,omitempty" json:"params,omitempty"`
}

var (
	idRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)

	families  = []string{"linux", "windows", "android", "macos", "ios"}
	profiles  = []string{"cloud-init", "cloudbase-init", "android", "tart", "phone"}
	readiness = []string{"ssh", "rdp", "adb", "none"}
	services  = []string{"ssh", "rdp", "vnc", "adb", "http", "https"}
	arches    = []string{"x86_64", "arm64"}
	formats   = []string{"qcow2", "raw", "img", "iso", "container", "oci", "none"}

	// URL schemes each driver accepts.
	driverSchemes = map[string][]string{
		"libvirt":     {"https", "local"},
		"docker-qemu": {"docker", "https", "local"},
		"lxc":         {"incus"},
		"android":     {"docker", "local", "https"},
		"orchard":     {"oci"},
		"usb-phone":   {""},
	}

	// Drivers each family may run on.
	familyDrivers = map[string][]string{
		"linux":   {"libvirt", "docker-qemu", "lxc"},
		"windows": {"libvirt", "docker-qemu"},
		"android": {"android", "libvirt", "usb-phone"},
		"macos":   {"orchard"},
		"ios":     {"usb-phone", "orchard"},
	}

	// Families each guest profile is valid for.
	profileFamilies = map[string][]string{
		"cloud-init":     {"linux"},
		"cloudbase-init": {"windows"},
		"android":        {"android"},
		"tart":           {"macos", "ios"},
		"phone":          {"android", "ios"},
	}

	desktopServices = []string{"rdp", "vnc"}
)

// scheme returns the URL scheme of an image reference ("" if empty).
func scheme(url string) string {
	switch {
	case url == "":
		return ""
	case strings.HasPrefix(url, "https://"):
		return "https"
	case strings.HasPrefix(url, "local:"):
		return "local"
	case strings.HasPrefix(url, "docker://"):
		return "docker"
	case strings.HasPrefix(url, "oci://"):
		return "oci"
	case strings.HasPrefix(url, "incus:"):
		return "incus"
	}
	return "invalid"
}

// Validate checks a template against docs/FORMAT.md. All problems are returned joined.
func (t *Template) Validate() error {
	var errs []error
	add := func(format string, a ...any) { errs = append(errs, fmt.Errorf(format, a...)) }

	if !idRe.MatchString(t.ID) {
		add("id %q must match %s", t.ID, idRe)
	}
	if t.Name == "" {
		add("name is required")
	}
	if t.Version == "" {
		add("version is required")
	}
	if !slices.Contains(families, t.Family) {
		add("family %q must be one of %v", t.Family, families)
	}
	if !slices.Contains(profiles, t.GuestProfile) {
		add("guest_profile %q must be one of %v", t.GuestProfile, profiles)
	} else if !slices.Contains(profileFamilies[t.GuestProfile], t.Family) {
		add("guest_profile %q is not valid for family %q", t.GuestProfile, t.Family)
	}
	if !slices.Contains(readiness, t.Readiness) {
		add("readiness %q must be one of %v", t.Readiness, readiness)
	}
	if t.DefaultUser == "" && t.GuestProfile != "phone" {
		add("default_user is required")
	}

	if len(t.SupportedServices) == 0 && t.GuestProfile != "phone" {
		add("supported_services must not be empty")
	}
	seen := map[string]bool{}
	for _, s := range t.SupportedServices {
		if !slices.Contains(services, s) {
			add("unknown service %q (known: %v)", s, services)
		}
		if seen[s] {
			add("duplicate service %q", s)
		}
		seen[s] = true
		if slices.Contains(desktopServices, s) && !slices.Contains([]string{"linux", "windows", "macos"}, t.Family) {
			add("service %q is only valid for desktop families", s)
		}
		if s == "adb" && t.Family != "android" {
			add("service adb is only valid for family android")
		}
	}
	if (t.Readiness == "ssh" || t.Readiness == "rdp" || t.Readiness == "adb") && !seen[t.Readiness] {
		add("readiness %q requires that service in supported_services", t.Readiness)
	}
	for s, p := range t.ServicePorts {
		if !seen[s] {
			add("service_ports: %q is not in supported_services", s)
		}
		if p < 1 || p > 65535 {
			add("service_ports: %q port %d out of range", s, p)
		}
	}

	if len(t.Arch) == 0 {
		add("arch must not be empty")
	}
	for _, a := range t.Arch {
		if !slices.Contains(arches, a) {
			add("arch %q must be one of %v", a, arches)
		}
	}

	if len(t.Images) == 0 {
		add("images must contain at least one driver")
	}
	resourceBased := false
	for d, img := range t.Images {
		allowed, ok := driverSchemes[d]
		if !ok {
			add("images: unknown driver %q", d)
			continue
		}
		if d != "usb-phone" {
			resourceBased = true
		}
		if fam, ok := familyDrivers[t.Family]; ok && !slices.Contains(fam, d) {
			add("images: driver %q cannot run family %q", d, t.Family)
		}
		sc := scheme(img.URL)
		if !slices.Contains(allowed, sc) {
			add("images.%s: url %q has scheme %q, allowed: %q", d, img.URL, sc, allowed)
		}
		if !slices.Contains(formats, img.Format) {
			add("images.%s: format %q must be one of %v", d, img.Format, formats)
		}
		if img.SHA256 != "" {
			if b, err := hex.DecodeString(img.SHA256); err != nil || len(b) != 32 {
				add("images.%s: sha256 must be 64 hex characters", d)
			}
		}
	}

	checkRes := func(name string, r Resources) {
		if r.VCPUs < 0 || r.RAMMB < 0 || r.StorageGB < 0 {
			add("%s must not be negative", name)
		}
		if resourceBased && (r.VCPUs < 1 || r.RAMMB < 1) {
			add("%s: vcpus and ram_mb must be at least 1", name)
		}
	}
	checkRes("min_resources", t.MinResources)
	checkRes("default_resources", t.DefaultResources)
	if t.MinResources.VCPUs > t.DefaultResources.VCPUs ||
		t.MinResources.RAMMB > t.DefaultResources.RAMMB ||
		t.MinResources.StorageGB > t.DefaultResources.StorageGB {
		add("min_resources must not exceed default_resources")
	}

	if t.License && t.Family != "windows" {
		add("license is only valid for family windows")
	}
	return errors.Join(errs...)
}

// LoadDir reads and validates all templates/*.yaml files in dir.
func LoadDir(dir string) ([]Template, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no templates found in %s", dir)
	}
	var (
		out  []Template
		errs []error
		ids  = map[string]string{}
	)
	for _, f := range files {
		t, err := loadFile(f)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", f, err))
			continue
		}
		if want := strings.TrimSuffix(filepath.Base(f), ".yaml"); t.ID != want {
			errs = append(errs, fmt.Errorf("%s: id %q must match file name %q", f, t.ID, want))
		}
		if prev, dup := ids[t.ID]; dup {
			errs = append(errs, fmt.Errorf("%s: duplicate id %q (also in %s)", f, t.ID, prev))
		}
		ids[t.ID] = f
		out = append(out, t)
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func loadFile(path string) (Template, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Template{}, err
	}
	return parse(raw)
}

func parse(raw []byte) (Template, error) {
	var t Template
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true)
	if err := dec.Decode(&t); err != nil {
		return Template{}, err
	}
	if err := t.Validate(); err != nil {
		return Template{}, err
	}
	return t, nil
}
