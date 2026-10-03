package distrokind

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/opencharly/spec/schema"
	"github.com/opencharly/spec/schemaconcat"
)

// This plugin serves `kind: distro`, and it validates an authored `distro:` body against
// its OWN self-contained #DistroInput before dispatching. That duplication is sanctioned —
// the plugin schema must compile standalone while spec's #Distro must generate Go — but it
// means the two can DRIFT, and when they do the failure is confusing: charly's core
// accepts a field and this plugin rejects it with `#DistroInput.<field>: field not allowed`,
// which reads like the field does not exist at all.
//
// That is not hypothetical, and it has now happened three times. spec gained
// `#Distro.installer` and this def did not, so an authored installer: block was rejected
// here for months. `disk_layout` was about to repeat it. And spec's #Format gained
// `present_template`, which drifted SILENTLY because the guard compared TOP-LEVEL fields
// only — the field lives on the inner #DsFormat, so nothing noticed.
//
// So this compares against spec's OWN CUE source — the embed in the spec module this plugin
// already requires — instead of a hand-maintained list that can silently lag spec, and it
// descends into every mirrored inner def. Field NAMES are compared, never bodies: the inner
// defs deliberately differ in name (#DsBootloader vs #Bootloader) and in @go() annotations,
// and normalising the name strips exactly that noise while still catching a dropped field.
func TestEveryMirroredDefMatchesSpec(t *testing.T) {
	specSrc := specSchemaSource(t)
	local, err := os.ReadFile(filepath.Join("schema", "distro.cue"))
	if err != nil {
		t.Fatalf("reading this plugin's schema: %v", err)
	}
	src := string(local)

	// The authored top level must authorise exactly what spec's #Distro does.
	assertSameFields(t, "#DistroInput", topLevelFields(t, src, "#DistroInput"),
		"#Distro", topLevelFields(t, specSrc, "#Distro"))

	// Every inner def this schema mirrors is named "#Ds" + spec's def name; spec's two
	// distro-scoped installer defs carry an extra "Distro" segment.
	for _, name := range mirroredDefs(t, src) {
		trimmed := strings.TrimPrefix(name, "#Ds")
		specName := "#" + trimmed
		if !strings.Contains(specSrc, specName+": {") {
			specName = "#Distro" + trimmed
		}
		if !strings.Contains(specSrc, specName+": {") {
			t.Errorf("this schema defines %s, but spec defines neither %s nor #Distro%s",
				name, "#"+trimmed, trimmed)
			continue
		}
		assertSameFields(t, name, topLevelFields(t, src, name),
			specName, topLevelFields(t, specSrc, specName))
	}
}

// specSchemaSource is spec's whole CUE schema, concatenated from the embed in the spec
// module this plugin requires — the same FS charly core splices at runtime.
func specSchemaSource(t *testing.T) string {
	t.Helper()
	src, files, err := schemaconcat.ConcatSchema(schema.FS, ".", nil)
	if err != nil {
		t.Fatalf("concatenating spec's embedded schema: %v", err)
	}
	if src == "" || len(files) == 0 {
		t.Fatal("spec's embedded schema concatenated to nothing — the guard would pass vacuously")
	}
	return src
}

func assertSameFields(t *testing.T, localName string, got []string, specName string, want []string) {
	t.Helper()
	if len(got) == 0 || len(want) == 0 {
		t.Errorf("%s or %s parsed to ZERO fields — the extractor, not the schema, is broken",
			localName, specName)
		return
	}
	if strings.Join(got, ",") == strings.Join(want, ",") {
		return
	}
	t.Errorf("%s fields drifted from spec's %s.\n got: %v\nwant: %v\n"+
		"If spec added a field, mirror it here; if spec removed one, remove it here.",
		localName, specName, got, want)
}

// mirroredDefs returns every "#Ds*" def this schema defines, sorted.
func mirroredDefs(t *testing.T, src string) []string {
	t.Helper()
	var out []string
	for _, m := range regexp.MustCompile(`(?m)^(#Ds[A-Za-z0-9_]+):\s*\{`).FindAllStringSubmatch(src, -1) {
		out = append(out, m[1])
	}
	sort.Strings(out)
	if len(out) == 0 {
		t.Fatal("no #Ds* defs found — the extractor is broken and the guard would pass vacuously")
	}
	return out
}

// Every field #DistroInput declares must reference a def that actually exists in this
// file, or the schema compiles into a dangling reference that only fails at use.
func TestDistroInputReferencedDefsExist(t *testing.T) {
	local, err := os.ReadFile(filepath.Join("schema", "distro.cue"))
	if err != nil {
		t.Fatalf("reading this plugin's schema: %v", err)
	}
	src := string(local)
	defined := map[string]bool{}
	for _, m := range regexp.MustCompile(`(?m)^(#[A-Za-z0-9_]+):`).FindAllStringSubmatch(src, -1) {
		defined[m[1]] = true
	}
	body := defBody(t, src, "#DistroInput")
	for _, m := range regexp.MustCompile(`(#Ds[A-Za-z0-9_]+)`).FindAllStringSubmatch(body, -1) {
		if !defined[m[1]] {
			t.Errorf("#DistroInput references %s, which is not defined in this schema", m[1])
		}
	}
}

func defBody(t *testing.T, src, name string) string {
	t.Helper()
	i := strings.Index(src, name+": {")
	if i < 0 {
		t.Fatalf("%s is not defined", name)
	}
	rest := src[i:]
	j := strings.Index(rest, "\n}")
	if j < 0 {
		t.Fatalf("%s is not closed", name)
	}
	return rest[:j]
}

func topLevelFields(t *testing.T, src, name string) []string {
	t.Helper()
	var out []string
	for _, line := range strings.Split(defBody(t, src, name), "\n")[1:] {
		m := regexp.MustCompile(`^\t([a-z_]+)\??:`).FindStringSubmatch(line)
		if m != nil {
			out = append(out, m[1])
		}
	}
	sort.Strings(out)
	return out
}
