package distrokind

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"

	"github.com/opencharly/spec/spec"
)

// The SECOND parity guard, and it covers a different seam from schema_parity_test.go.
//
// That one compares the SERVED SCHEMA against spec — it catches "charly accepts a field
// and this plugin rejects it". This one compares the RESOLVER against spec — it catches
// the strictly worse failure where everything accepts the field, validates cleanly, and
// then resolveDistro's hand-written struct literal quietly forgets to carry it. Nothing
// errors. The consumer just sees nil and takes its default path.
//
// That is not hypothetical either. disk_layout and installer were both added to spec's
// #Distro and to this plugin's #DistroInput, and both were dropped here. The visible
// effect of the disk_layout drop was that the btrfs-subvolume feature — landed across
// spec#72, plugin-vm#6, plugin-vm#7 and charly#476 — did nothing at all: every distro
// declaring subvolumes silently got the historical bare-filesystem layout, and every one
// of those PRs was green.
//
// So the schema guard alone was not enough, and adding a field to the schema is exactly
// the moment someone forgets the resolver.
func TestResolveDistro_CarriesEveryResolvedField(t *testing.T) {
	// An authored distro with EVERY field set to something non-zero and distinguishable.
	// Fields are set through JSON rather than a Go literal so this test exercises the same
	// decode resolveDistro itself performs.
	authored := map[string]any{
		"inherits":         "arch",
		"inherit_packages": true,
		"version":          "42",
		"bootstrap":        map[string]any{"install_cmd": "pacman -S"},
		"workaround":       []string{"no-check-certificate"},
		"format":           map[string]any{"pac": map[string]any{"install_cmd": "pacman -S"}},
		"base_user":        map[string]any{"name": "user", "uid": 1000, "gid": 1000, "home": "/home/user"},
		"pacstrap":         map[string]any{"base_package": []string{"base"}},
		"debootstrap":      map[string]any{"suite": "trixie"},
		"alpine_bootstrap": map[string]any{"branch": "edge"},
		"bootloader":       map[string]any{"install_template": "echo install"},
		"disk_layout":      map[string]any{"esp_mount_point": "/boot"},
		"installer": map[string]any{
			"volume_id": "cidata",
			"file":      []any{map[string]any{"path": "answers.json", "content": "{}"}},
		},
		"dnf": map[string]any{"install_cmd": "dnf install"},
	}
	raw, err := json.Marshal(authored)
	if err != nil {
		t.Fatalf("marshalling the authored distro: %v", err)
	}

	reply, err := resolveDistro(spec.DistroResolveInput{Distro: raw})
	if err != nil {
		t.Fatalf("resolveDistro: %v", err)
	}
	if reply.Resolved == nil {
		t.Fatal("resolveDistro returned no resolved distro")
	}

	// Every exported field of ResolvedDistro must be non-zero. Raw is populated from the
	// input; the rest must have been CARRIED. A field left at its zero value means the
	// struct literal in resolve.go forgot it.
	rv := reflect.ValueOf(*reply.Resolved)
	rt := rv.Type()
	var dropped []string
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if !f.IsExported() {
			continue
		}
		if rv.Field(i).IsZero() {
			dropped = append(dropped, f.Name)
		}
	}
	sort.Strings(dropped)
	if len(dropped) > 0 {
		t.Fatalf("resolveDistro dropped %d field(s) that spec.ResolvedDistro declares: %v\n"+
			"Every one of these was authored in the input above and validated cleanly, then "+
			"silently arrived nil at the consumer.\n"+
			"Add them to the struct literal in resolve.go. If a field genuinely must not be "+
			"carried, say why HERE — do not delete this test.", len(dropped), dropped)
	}
}

// The two fields whose drop this test was written for, asserted by value rather than by
// non-zero-ness, so a future refactor cannot satisfy the guard above with a wrong value.
func TestResolveDistro_DiskLayoutAndInstallerArriveIntact(t *testing.T) {
	raw, err := json.Marshal(map[string]any{
		"disk_layout": map[string]any{
			"esp_mount_point": "/boot",
			"subvolume": []any{
				map[string]any{"name": "@", "mount_point": "/", "mount_options": "compress=zstd"},
				map[string]any{"name": "@home", "mount_point": "/home"},
			},
		},
		"installer": map[string]any{
			"volume_id": "cidata",
			"file": []any{
				map[string]any{"path": "user_configuration.json", "content": `{"hostname":"{{.Hostname}}"}`},
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	reply, err := resolveDistro(spec.DistroResolveInput{Distro: raw})
	if err != nil {
		t.Fatalf("resolveDistro: %v", err)
	}
	got := reply.Resolved

	if got.DiskLayout == nil {
		t.Fatal("disk_layout did not survive resolve — the btrfs subvolume feature is inert without it")
	}
	if got.DiskLayout.EspMountPoint != "/boot" {
		t.Errorf("esp_mount_point\n got: %q\nwant: /boot", got.DiskLayout.EspMountPoint)
	}
	if len(got.DiskLayout.Subvolume) != 2 {
		t.Fatalf("want 2 subvolumes, got %d", len(got.DiskLayout.Subvolume))
	}
	if got.DiskLayout.Subvolume[0].Name != "@" || got.DiskLayout.Subvolume[0].MountPoint != "/" {
		t.Errorf("first subvolume corrupted: %+v", got.DiskLayout.Subvolume[0])
	}

	if got.Installer == nil {
		t.Fatal("installer did not survive resolve — an iso VM cannot render its answers without it")
	}
	if got.Installer.VolumeID != "cidata" {
		t.Errorf("volume_id\n got: %q\nwant: cidata", got.Installer.VolumeID)
	}
	if len(got.Installer.Files) != 1 || got.Installer.Files[0].Path != "user_configuration.json" {
		t.Fatalf("installer files corrupted: %+v", got.Installer.Files)
	}
	// The template body must arrive UNRENDERED — rendering happens later, against the vm
	// entity's data. A resolver that touched it would break every seed.
	if got.Installer.Files[0].Content != `{"hostname":"{{.Hostname}}"}` {
		t.Errorf("template body was altered in transit: %q", got.Installer.Files[0].Content)
	}
}
