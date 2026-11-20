package distrokind

// provider_declaration_test.go — the guard for the TWO declarations a kind needs.
//
// A kind is SERVABLE through NewMeta's {Class:"kind", Word:…, InputDef:…} capability and
// AUTHORABLE through this candy's `plugin: providers:` list. They are read by different code:
// NewMeta by the provider registry, the manifest list by charly/plugin_prescan.go:725 into
// `declaredKind`, which becomes spec.Threaded.Kinds — the ONLY recognition path now that
// spec.KindWords is empty.
//
// #16 and #18 shipped capabilities without the matching manifest rows, so both kinds were
// rejected at PARSE time ("no kind discriminator") while every test passed: Invoke and Describe
// never touch the parser. This test is what makes the two lists one fact.

import (
	"context"
	"os"
	"sort"
	"strings"
	"testing"

	pb "github.com/opencharly/spec/proto"
	"gopkg.in/yaml.v3"
)

// TestManifestDeclaresExactlyTheAdvertisedKindWords asserts the manifest's provider list is
// precisely the set of kind words NewMeta advertises — no missing row (unparseable) and no
// extra one (a word the provider cannot decode).
func TestManifestDeclaresExactlyTheAdvertisedKindWords(t *testing.T) {
	advertised := advertisedKindWords(t)
	declared := declaredKindWords(t)

	if len(advertised) == 0 {
		t.Fatal("NewMeta advertises no kind capability — the extractor, not the manifest, is broken")
	}
	if len(declared) == 0 {
		t.Fatal("the manifest declares no kind provider — the extractor, not NewMeta, is broken")
	}

	// MISSING: advertised but not declared. The kind is servable and unparseable — the exact
	// defect this PR fixes, and the one no schema or Invoke test can see.
	for _, w := range advertised {
		if !contains(declared, w) {
			t.Errorf("NewMeta advertises kind %q but the candy manifest does not declare "+
				"`- kind:%s`. an authored %s: node would be rejected at parse with "+
				"\"no kind discriminator\", while every Invoke and Describe test passed",
				w, w, w)
		}
	}
	// EXTRA: declared but not advertised. The host would accept the word and then find no
	// provider to decode it.
	for _, w := range declared {
		if !contains(advertised, w) {
			t.Errorf("the manifest declares `- kind:%s` but NewMeta advertises no such kind — "+
				"the parser would accept the word and no provider could decode it", w)
		}
	}
}

// advertisedKindWords is every Word NewMeta advertises with Class "kind", read from Describe —
// the reply a host resolves a kind from.
func advertisedKindWords(t *testing.T) []string {
	t.Helper()
	caps, err := NewMeta().Describe(context.Background(), &pb.Empty{})
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	var out []string
	for _, c := range caps.GetProvided() {
		if c.GetClass() == "kind" {
			out = append(out, c.GetWord())
		}
	}
	sort.Strings(out)
	return out
}

// declaredKindWords is every `kind:<word>` row in this candy's own `plugin: providers:` list —
// the list charly/plugin_prescan.go reads.
func declaredKindWords(t *testing.T) []string {
	t.Helper()
	b, err := os.ReadFile("charly.yml")
	if err != nil {
		t.Fatalf("reading this candy's charly.yml: %v", err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(b, &doc); err != nil {
		t.Fatalf("parsing charly.yml: %v", err)
	}
	// The file's single top-level key is the candy entity; its `candy: plugin: providers:` list
	// is what the prescan walks.
	var out []string
	var walk func(any)
	walk = func(v any) {
		m, ok := v.(map[string]any)
		if !ok {
			return
		}
		if pv, ok := m["plugin"].(map[string]any); ok {
			if provs, ok := pv["providers"].([]any); ok {
				for _, p := range provs {
					if s, ok := p.(string); ok {
						if w, found := strings.CutPrefix(s, "kind:"); found {
							out = append(out, w)
						}
					}
				}
			}
		}
		for _, child := range m {
			walk(child)
		}
	}
	walk(any(doc))
	sort.Strings(out)
	return out
}
