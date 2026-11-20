package distrokind

// source_parity_test.go — the guard the `source.cue` header promised.
//
// WHY THIS FILE EXISTS: an earlier head carried a comment saying "os_parity-style coverage for
// them lives in source_parity_test.go" while no such file existed and the source tests were in
// invoke_load_test.go, testing the RUNTIME arm. The four helper defs — #SourceImageRef,
// #SourceDownload, #SourceIso, #SourceBootstrap — had no guard against spec at all, which is
// the same gap #OsInput's four inner defs had and for the same reason: this package's
// TestEveryMirroredDefMatchesSpec follows the "#Ds" + spec-name convention, and these carry
// names of their own.
//
// The failure this prevents is documented three times in schema_parity_test.go's header: when
// the plugin's mirror of spec's authored surface drifts, core accepts a field and the plugin
// refuses it with `field does not exist`. A guard written after the first confusion is worth
// less than one written before it.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSourceInnerDefsMatchSpec guards the four helper defs against spec's, by FIELDS.
//
// The mapping is spelled out rather than derived because the names genuinely differ — a
// mechanical "#Source" + spec-name rule would look for "#SourceDownload: {" and find it, but
// would also look for "#SourceSourceIso" and silently find nothing.
func TestSourceInnerDefsMatchSpec(t *testing.T) {
	specSrc := specSchemaSource(t)
	local, err := os.ReadFile(filepath.Join("schema", "source.cue"))
	if err != nil {
		t.Fatalf("reading this plugin's source schema: %v", err)
	}
	src := string(local)

	for _, pair := range []struct{ local, spec string }{
		{"#SourceDownload", "#Download"},
		{"#SourceIso", "#SourceIso"},
		{"#SourceBootstrap", "#SourceBootstrap"},
	} {
		if !strings.Contains(specSrc, pair.spec+": {") && !strings.Contains(specSrc, pair.spec+":\t{") {
			t.Errorf("spec no longer defines %s — this package's %s has nothing to mirror",
				pair.spec, pair.local)
			continue
		}
		assertSameFields(t, pair.local, defFields(t, src, pair.local),
			pair.spec, defFields(t, specSrc, pair.spec))
	}

	// #SourceImageRef is a SCALAR regex, not a struct: it is compared by the shape it admits,
	// because a field comparison of two regexes is meaningless. The property that matters is
	// D-SRC-1's — a digest, when present, is pinned by the same sha256 grammar as #Download's.
	if !strings.Contains(defShape(src), "sha256:[0-9a-f]{64}") {
		t.Error("#SourceImageRef lost its sha256 digest arm — an oci origin could then name an " +
			"unpinned remote, which D-SRC-1 forbids for every remote origin")
	}
}

// TestSourceAltHasExactlySpecsFourOrigins proves the union's ARMS are spec's four, so a source
// authored against this plugin cannot name an origin the contract does not define — and cannot
// omit one the contract does.
func TestSourceAltHasExactlySpecsFourOrigins(t *testing.T) {
	specSrc := specSchemaSource(t)
	local, err := os.ReadFile(filepath.Join("schema", "source.cue"))
	if err != nil {
		t.Fatalf("reading this plugin's source schema: %v", err)
	}
	src := string(local)

	for _, origin := range []string{"oci", "disk", "iso", "bootstrap"} {
		// The ARM must exist here with the origin's own key required inside it.
		if !strings.Contains(src, "\t"+origin+": {") || !strings.Contains(src, origin+"!:") {
			t.Errorf("#SourceAlt has no %q arm carrying a required %s!: key — an authored "+
				"%s origin would be rejected here", origin, origin, origin)
		}
		// And spec must still declare it, or this plugin offers a word core does not know.
		if !strings.Contains(specSrc, "\t"+origin+":") {
			t.Errorf("#SourceAlt declares the %q origin, which spec's #SourceAlt no longer does",
				origin)
		}
	}

	// EXACTLY four, both sides: a fifTH arm here is a word core refuses.
	arms := 0
	for _, origin := range []string{"oci", "disk", "iso", "bootstrap"} {
		if strings.Contains(src, "\t"+origin+": {\n\t\t"+origin+"!:") {
			arms++
		}
	}
	if arms != 4 {
		t.Errorf("#SourceAlt has %d well-formed origin arms, want exactly 4 (oci, disk, iso, "+
			"bootstrap) — D-SRC-1 admits exactly one origin, and the union is what enforces it",
			arms)
	}
}

// TestSourceRequiresOsAndDescription asserts D-SRC-2 on every arm: a source that does not name
// its os cannot say what converts it into the other form (§16.4).
func TestSourceRequiresOsAndDescription(t *testing.T) {
	src := defShape(string(mustRead(t, "schema/source.cue")))

	// #SourceAlt must carry os! and description! inside every arm — checked on the source text
	// between the union's opening and its close, so an arm that dropped one is caught.
	union := src[strings.Index(src, "#SourceAlt: {"):]
	for _, origin := range []string{"oci", "disk", "iso", "bootstrap"} {
		i := strings.Index(union, "\t"+origin+": {")
		if i < 0 {
			t.Fatalf("no %q arm in #SourceAlt", origin)
		}
		arm := union[i:]
		if j := strings.Index(arm, "\t}\n"); j > 0 {
			arm = arm[:j]
		}
		if !strings.Contains(arm, "os!:") {
			t.Errorf("the %q arm does not require os! — D-SRC-2 makes every source name the os "+
				"that owns its conversions", origin)
		}
		if !strings.Contains(arm, "description!:") {
			t.Errorf("the %q arm does not require description! — the node would be unreadable "+
				"in an authored charly.yml", origin)
		}
	}
}

// mustRead reads a schema file or fails the test.
func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(path))
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return b
}

// defShape returns the whole schema source minus its comments, so a comment quoting a field
// cannot satisfy an assertion about the definition.
func defShape(src string) string {
	var b strings.Builder
	for _, line := range strings.Split(src, "\n") {
		if t := strings.TrimSpace(line); strings.HasPrefix(t, "//") {
			continue
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}
