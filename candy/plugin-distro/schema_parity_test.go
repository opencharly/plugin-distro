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

// TestOsInputAddsExactlyOsFields is the same guard for the SECOND word this plugin serves.
//
// It exists because the failure mode is already documented three times over in this file's
// header: when the plugin's mirror of spec's authored surface drifts, core accepts a field
// and the plugin refuses it with `field not allowed`, which reads like the field does not
// exist. `os:` is a NEW word, so it has no drift yet — and this test is what keeps it that
// way, from the first commit rather than after the first confusing rejection.
//
// The comparison is against spec's #Os, added in opencharly/spec#402. #OsInput's shared half
// is a field-for-field mirror of #Distro, so it is checked by TestEveryMirroredDefMatchesSpec
// above; what is NEW here is the five fields #Os carries and #Distro does not, which is
// exactly the surface that has no other guard.
func TestOsInputAddsExactlyOsFields(t *testing.T) {
	specSrc := specSchemaSource(t)
	local, err := os.ReadFile(filepath.Join("schema", "os.cue"))
	if err != nil {
		t.Fatalf("reading this plugin's os schema: %v", err)
	}

	// defFields, NOT topLevelFields: the shared helper's regex is `^\t([a-z_]+)\??:` and does
	// not match a REQUIRED field, so `description!`, `family!`, `package_manager!` and `init!`
	// extract as nothing on BOTH sides — zero-vs-zero, which PASSES. With topLevelFields this
	// test reported agreement while the four fields it exists to compare were invisible, and
	// deleting `init!` from #OsInput failed no test at all. defFields is the sibling this PR
	// added for exactly this, and the outer comparison needed it as much as the inner one.
	osFields := defFields(t, string(local), "#OsInput")
	specOs := defFields(t, specSrc, "#Os")

	// The required half must be visibly present, or the comparison below is vacuous: assert
	// the four D-CANDY-3 / D-SRC-2 fields were actually READ.
	for _, required := range []string{"description", "family", "package_manager", "init"} {
		if !contains(specOs, required) {
			t.Fatalf("spec's #Os field %q did not extract — the extractor is broken, not the "+
				"schema, and the comparison below would pass vacuously", required)
		}
	}

	// Every field spec's #Os declares must exist here. A MISSING field means an authored
	// `os:` body core validates and this plugin then rejects.
	for _, f := range specOs {
		if !contains(osFields, f) {
			t.Errorf("#OsInput does not mirror spec's #Os field %q — core would accept an "+
				"authored `os:` body this plugin then refuses", f)
		}
	}

	// And nothing ELSE: a field only this def has is a word the contract does not define, so
	// it would be accepted here and rejected by core — the same drift from the other side.
	for _, f := range osFields {
		if !contains(specOs, f) {
			t.Errorf("#OsInput declares %q, which spec's #Os does not — an authored `os:` body "+
				"would pass this plugin and fail core", f)
		}
	}
}

// contains reports whether list holds want.
func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

// TestOsOnlyInnerDefsMatchSpec guards the four inner defs #OsInput introduces, which
// TestEveryMirroredDefMatchesSpec does NOT reach.
//
// That test mirrors this package's "#Ds" + spec-name convention and compares those defs
// against spec. #OsPackageManager, #OsInit, #OsBoot and #OsBootstrapper follow no such
// convention — they are this file's own names for shapes spec calls
// _packageManager / #OsInit / #Boot / #OsBootstrapper — so the mirrored-defs sweep skips
// them entirely and they would drift unguarded.
//
// The mapping is spelled out rather than derived because the names genuinely differ: a
// mechanical "#Os" + spec-name rule would look for #OsOsInit and find nothing, which is a
// silent pass, not a guard.
func TestOsOnlyInnerDefsMatchSpec(t *testing.T) {
	specSrc := specSchemaSource(t)
	local, err := os.ReadFile(filepath.Join("schema", "os.cue"))
	if err != nil {
		t.Fatalf("reading this plugin's os schema: %v", err)
	}
	src := string(local)

	for _, pair := range []struct{ local, spec string }{
		{"#OsBoot", "#Boot"},
		{"#OsBootstrapper", "#OsBootstrapper"},
	} {
		if !strings.Contains(specSrc, pair.spec+": {") {
			t.Errorf("spec no longer defines %s — this package's %s has nothing to mirror",
				pair.spec, pair.local)
			continue
		}
		assertSameFields(t, pair.local, defFields(t, src, pair.local),
			pair.spec, defFields(t, specSrc, pair.spec))
	}

	// #OsInit is NOT compared as fields, because spec's is a COMPREHENSION:
	// `#OsInit: {for r, _ in _initRole {(r)!: #Ref @ref(init)}}` has no static fields at all,
	// so a field-for-field comparison against it is zero-versus-two and proves nothing. What
	// the comprehension PRODUCES is one key per _initRole row, so that is what this asserts:
	// the two roles exist here, and nothing else does.
	initRows := declLine(specSrc, "_initRole:")
	for _, role := range []string{"container", "system"} {
		if !strings.Contains(initRows, role) {
			t.Errorf("spec's _initRole no longer has the %q role this #OsInit declares", role)
		}
	}
	localInit := defFields(t, src, "#OsInit")
	for _, f := range localInit {
		if !strings.Contains(initRows, f) {
			t.Errorf("#OsInit declares role %q, which spec's _initRole does not — spec's #OsInit "+
				"is a comprehension over that table, so this key would be a role core cannot name", f)
		}
	}
	// Count the table's ROWS from its own text: `_initRole: {container: {}, system: {}}` is
	// two. Counting ':' and halving is wrong — the key itself contributes one.
	// A row of that table is `word: {}` — counted WITHOUT a leading brace, because the second
	// and later rows are separated by `, ` rather than by their own `{`.
	roleCount := len(regexp.MustCompile(`[a-z_]+: \{\}`).FindAllString(initRows, -1))
	if len(localInit) != roleCount {
		t.Errorf("#OsInit declares %d role(s) %v; spec's _initRole declares %d — the two disagree",
			len(localInit), localInit, roleCount)
	}

	// #OsPackageManager is a WORD SET, not a struct, so it is compared as words rather than
	// as fields: every manager it admits must be one spec's _packageManager declares, or an
	// authored `package_manager:` would pass this gate and fail the contract.
	for _, word := range []string{"dnf", "apt", "pacman", "apk", "zypper"} {
		if !strings.Contains(src, `"`+word+`"`) {
			t.Errorf("#OsPackageManager does not admit %q — spec's _packageManager does", word)
		}
	}
	// Read each set from its DECLARATION LINE rather than from a brace-delimited body: spec's
	// `_packageManager: {dnf: {}, apt: {}, …}` closes an inner struct on the same line, so a
	// "text between the key and the first }" read returns only `dnf` and silently passes.
	specManagers := declLine(specSrc, "_packageManager:")
	osManagers := strings.TrimSpace(strings.TrimPrefix(declLine(src, "#OsPackageManager:"), "#OsPackageManager:"))
	for _, word := range strings.Fields(strings.NewReplacer(`"`, " ", `|`, " ", "*", " ").Replace(osManagers)) {
		if !strings.Contains(specManagers, word) {
			t.Errorf("#OsPackageManager admits %q, which spec's _packageManager does not", word)
		}
	}
}

// defFields is topLevelFields' brace- and attribute-aware sibling, for the defs the shared
// helper cannot read.
//
// topLevelFields matches `^\t([a-z_]+)\??:`, which misses two shapes that are perfectly
// valid CUE and both appear here: a REQUIRED field (`container!:`) and a def written on ONE
// LINE (`#Boot: {firmware: …}`). It returns zero fields for either — and zero compared with
// zero PASSES, so a guard built naively on it would report agreement between two defs it
// never read.
func defFields(t *testing.T, src, name string) []string {
	t.Helper()
	// Exact key match, NOT defBody: that helper finds the name as a SUBSTRING, so looking up
	// "#Os" in a source that also declares "#OsInit" returns the wrong def and compares two
	// unrelated field lists.
	lines := strings.Split(src, "\n")
	start := -1
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), name+":") {
			start = i
			break
		}
	}
	if start < 0 {
		t.Fatalf("%s not found in the schema source", name)
	}
	// Take whole lines until the def's braces close: a one-line def ends on its own line, a
	// multi-line one runs until depth returns to zero.
	body := ""
	depth0 := 0
	for _, line := range lines[start:] {
		body += line + "\n"
		depth0 += strings.Count(line, "{") - strings.Count(line, "}")
		if depth0 <= 0 && strings.Contains(line, "}") {
			break
		}
	}
	var out []string
	fieldRE := regexp.MustCompile(`(?:^|[,{])\s*([a-z_]+)[?!]?:`)
	depth := 0
	for i, line := range strings.Split(body, "\n") {
		// Only the def's OWN level: a nested block's keys are not this def's fields.
		scan := line
		if i == 0 {
			// A def written on one line carries its fields after the opening brace, e.g.
			// `#Boot: {firmware: …, kernel?: …}`. Scanning from the line start is enough —
			// the key itself has no `:`-terminated field shape the regex would match.
		}
		if depth == 1 || i == 0 {
			for _, m := range fieldRE.FindAllStringSubmatch(scan, -1) {
				out = append(out, m[1])
			}
		}
		depth += strings.Count(line, "{") - strings.Count(line, "}")
	}
	sort.Strings(out)
	return out
}

// declLine is the DECLARATION LINE of the definition or table whose key is `key` — used for
// vocabularies and one-line defs, where reading a brace-delimited body stops at the first
// inner `}` and returns a fragment. Returns "" when the key is absent.
func declLine(s, key string) string {
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), key) {
			return line
		}
	}
	return ""
}
