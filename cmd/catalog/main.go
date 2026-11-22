// Command catalog validates templates/*.yaml and generates catalog.json.
//
//	go run ./cmd/catalog          # validate + write catalog.json
//	go run ./cmd/catalog -check   # validate + fail if catalog.json is out of date
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"reflect"
	"time"
)

// SchemaVersion of catalog.json.
const SchemaVersion = 1

// Catalog is the generated catalog.json.
type Catalog struct {
	Schema      int        `json:"schema"`
	GeneratedAt string     `json:"generated_at"`
	Templates   []Template `json:"templates"`
}

func main() {
	dir := flag.String("dir", "templates", "directory with template YAML files")
	out := flag.String("out", "catalog.json", "catalog file to write / check")
	check := flag.Bool("check", false, "only check that the catalog file is up to date")
	flag.Parse()

	if err := run(*dir, *out, *check, time.Now()); err != nil {
		fmt.Fprintln(os.Stderr, "catalog:", err)
		os.Exit(1)
	}
}

func run(dir, out string, check bool, now time.Time) error {
	tpls, err := LoadDir(dir)
	if err != nil {
		return err
	}
	existing, err := readCatalog(out)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	upToDate := existing != nil && existing.Schema == SchemaVersion && sameTemplates(existing.Templates, tpls)

	if check {
		if !upToDate {
			return fmt.Errorf("%s is out of date, run: go run ./cmd/catalog", out)
		}
		fmt.Printf("%d templates valid, %s up to date\n", len(tpls), out)
		return nil
	}
	if upToDate {
		fmt.Printf("%d templates valid, %s unchanged\n", len(tpls), out)
		return nil
	}
	c := Catalog{Schema: SchemaVersion, GeneratedAt: now.UTC().Format(time.RFC3339), Templates: tpls}
	data, err := encode(c)
	if err != nil {
		return err
	}
	if err := os.WriteFile(out, data, 0o644); err != nil {
		return err
	}
	fmt.Printf("%d templates valid, wrote %s\n", len(tpls), out)
	return nil
}

func encode(c Catalog) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(c); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func readCatalog(path string) (*Catalog, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Catalog
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &c, nil
}

// sameTemplates compares via a JSON round trip so nil/empty differences don't matter.
func sameTemplates(a, b []Template) bool {
	norm := func(t []Template) any {
		raw, _ := json.Marshal(t)
		var v any
		_ = json.Unmarshal(raw, &v)
		return v
	}
	return reflect.DeepEqual(norm(a), norm(b))
}
