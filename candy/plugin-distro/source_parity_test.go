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
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	pb "github.com/opencharly/spec/proto"
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

	// EXACTLY four: the arms are ENUMERATED from the union's own body rather than counted over
	// the four names this test already knows. Counting the known names cannot detect a fifth —
	// which is what the previous version of this block did, while its comment claimed otherwise.
	got := unionArms(t, src, "#SourceAlt")
	want := map[string]bool{"oci": true, "disk": true, "iso": true, "bootstrap": true}
	for _, arm := range got {
		if !want[arm] {
			t.Errorf("#SourceAlt declares the %q arm, which spec's #SourceAlt does not — an "+
				"authored %s origin would pass this plugin and fail core", arm, arm)
		}
	}
	for arm := range want {
		found := false
		for _, g := range got {
			if g == arm {
				found = true
			}
		}
		if !found {
			t.Errorf("#SourceAlt is missing the %q arm", arm)
		}
	}
	if len(got) != 4 {
		t.Errorf("#SourceAlt declares %d arm(s) %v, want exactly 4 (oci, disk, iso, bootstrap) — "+
			"D-SRC-1 admits exactly one origin, and the union is what enforces it", len(got), got)
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

// unionArms enumerates the ARM NAMES of an S4 union def — the top-level `word: {` keys of its
// body, at the union's own nesting level.
//
// It exists because a test that loops over the names it already expects cannot detect a NEW one:
// it would pass with a fifth arm present, which is exactly what the comment above it claimed it
// would not.
func unionArms(t *testing.T, src, defName string) []string {
	t.Helper()
	lines := strings.Split(defShape(src), "\n")
	start := -1
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), defName+":") {
			start = i
			break
		}
	}
	if start < 0 {
		t.Fatalf("%s not found in the schema source", defName)
	}
	var arms []string
	depth := 0
	armRE := regexp.MustCompile(`^\t([a-z_]+): \{$`)
	for _, line := range lines[start:] {
		// At depth 1 we are inside the union body: a `word: {` there is an arm.
		if depth == 1 {
			if m := armRE.FindStringSubmatch(line); m != nil {
				arms = append(arms, m[1])
			}
		}
		depth += strings.Count(line, "{") - strings.Count(line, "}")
		if depth <= 0 && strings.Contains(line, "}") && len(arms) > 0 {
			break
		}
	}
	if len(arms) == 0 {
		t.Fatalf("%s parsed to ZERO arms — the extractor, not the schema, is broken", defName)
	}
	return arms
}

// TestNewMetaAdvertisesBothVocabularyWords is the capability-registration half, and it is the one
// a green suite otherwise cannot see.
//
// Every other test in this package calls provider{}.Invoke(...) DIRECTLY, bypassing NewMeta. So a
// misspelled Word ("sorce") or a wrong InputDef would make the kind unreachable from a real host
// and ship green: nothing would fail, because nothing would ask the plugin what it serves.
//
// It reads the reply a host reads — Describe — rather than the Go slice, because the slice is the
// input to a builder that could itself drop a row.
func TestNewMetaAdvertisesBothVocabularyWords(t *testing.T) {
	caps, err := NewMeta().Describe(context.Background(), &pb.Empty{})
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}

	// The word → InputDef mapping the host resolves a kind through.
	want := map[string]string{
		"distro": "#DistroInput",
		"os":     "#OsInput",
		"source": "#SourceInput",
	}
	got := map[string]string{}
	for _, c := range caps.GetProvided() {
		if c.GetClass() == "kind" {
			got[c.GetWord()] = c.GetInputDef()
		}
	}

	for word, def := range want {
		if _, ok := got[word]; !ok {
			t.Errorf("Describe does not advertise the %q kind — an authored %s: node would be "+
				"unreachable from a host, and every direct Invoke test would still pass",
				word, word)
			continue
		}
		if got[word] != def {
			t.Errorf("Describe advertises %q with InputDef %q, want %q — the host would resolve "+
				"the wrong gate def", word, got[word], def)
		}
	}

	// And nothing extra: a kind word this plugin does not implement is worse than a missing one,
	// because the host would route an authored body to a provider that cannot canonicalise it.
	for word := range got {
		if _, ok := want[word]; !ok {
			t.Errorf("Describe advertises the unexpected kind %q", word)
		}
	}

	// The schema must actually CARRY each gate def, or InputDef names a def the host cannot find.
	src := string(mustRead(t, "schema/source.cue")) + string(mustRead(t, "schema/os.cue")) +
		string(mustRead(t, "schema/distro.cue"))
	for _, def := range want {
		if !strings.Contains(src, def+":") {
			t.Errorf("Describe advertises InputDef %q but no schema file defines it", def)
		}
	}
}

// TestSourceInputOrListNamesExactlyTheFourArms inspects the GATE itself.
//
// #SourceInput is `or([#SourceAlt.oci, #SourceAlt.disk, #SourceAlt.iso, #SourceAlt.bootstrap])`.
// TestSourceAltHasExactlySpecsFourOrigins enumerates the UNION's arms, which proves the four are
// DEFINED — but nothing inspected the or([...]) that DECIDES which of them a value may be. Dropping
// `iso` from that list admits three origins instead of four, and until this test existed no
// assertion failed: the union still had four arms, and the gate silently offered fewer.
//
// "Four origins in both directions" is only true when this list is checked, so this is what makes
// the body's claim honest rather than half-proven.
func TestSourceInputOrListNamesExactlyTheFourArms(t *testing.T) {
	src := defShape(string(mustRead(t, "schema/source.cue")))

	// The or([...]) argument list, comment-stripped and whitespace-normalised.
	i := strings.Index(src, "#SourceInput:")
	if i < 0 {
		t.Fatal("no #SourceInput def")
	}
	rest := src[i+len("#SourceInput:"):]
	open := strings.Index(rest, "or([")
	if open < 0 {
		t.Fatal("#SourceInput is not an or([...]) — the gate admits one arm by construction, and " +
			"a different shape would have to be re-argued")
	}
	close := strings.Index(rest[open:], "])")
	if close < 0 {
		t.Fatal("the or([...]) is not closed")
	}
	list := rest[open+len("or([") : open+close]

	got := map[string]bool{}
	for _, m := range regexp.MustCompile(`#SourceAlt\.([a-z_]+)`).FindAllStringSubmatch(list, -1) {
		got[m[1]] = true
	}

	want := []string{"oci", "disk", "iso", "bootstrap"}
	for _, arm := range want {
		if !got[arm] {
			t.Errorf("#SourceInput's or([...]) does not name #SourceAlt.%s — that origin is "+
				"DEFINED but UNADMITTED, so an authored %s: body would be rejected by the gate "+
				"even though the union declares it", arm, arm)
		}
		delete(got, arm)
	}
	for arm := range got {
		t.Errorf("#SourceInput's or([...]) names #SourceAlt.%s, which is not one of the four "+
			"origins D-SRC-1 allows", arm)
	}
	if len(want) != 4 {
		t.Fatalf("the expected origin set is %d, not 4", len(want))
	}
}
