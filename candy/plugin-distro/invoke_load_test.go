package distrokind

// invoke_load_test.go — the runtime half of serving a SECOND kind word.
//
// The schema tests prove #OsInput mirrors spec's #Os. They say nothing about Invoke, and
// Invoke is where a two-word provider can silently lose data: the host dispatches
// kind-blindly and hands over a body it has ALREADY validated as complete, so decoding an
// `os:` body into spec.Distro would drop the five fields that make it an os and hand back a
// canonical form that is missing exactly what was just checked.
//
// That is not a hypothetical shape: it is the difference between canonicalising and
// truncating, and nothing else in this module would notice.

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/opencharly/sdk"
	pb "github.com/opencharly/spec/proto"
)

// osBody is a complete authored `os:` node: the five fields an os has and a distro does
// not, plus two from the shared build vocabulary so the test would catch a decode that
// dropped the SHARED half too.
const osBody = `{
	"description": "Fedora 43",
	"family": "fedora",
	"package_manager": "dnf",
	"init": {"system": "systemd", "container": "systemd"},
	"boot": {"firmware": "uefi"}
}`

// TestInvokeLoadKeepsTheOsOnlyFields proves the os arm decodes into spec.Os and returns
// every field, not just the ones spec.Distro happens to share.
func TestInvokeLoadKeepsTheOsOnlyFields(t *testing.T) {
	reply, err := provider{}.Invoke(context.Background(), &pb.InvokeRequest{
		Op:         sdk.OpLoad,
		Reserved:   "os",
		ParamsJson: []byte(osBody),
	})
	if err != nil {
		t.Fatalf("Invoke(OpLoad, os): %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(reply.GetResultJson(), &got); err != nil {
		t.Fatalf("decoding the reply: %v", err)
	}

	// The five an os has and a distro does not. A decode into spec.Distro loses all five,
	// which is the failure this test exists to make loud.
	for _, field := range []string{"description", "family", "package_manager", "init", "boot"} {
		if _, ok := got[field]; !ok {
			t.Errorf("OpLoad for word \"os\" dropped %q — the body was decoded into a type "+
				"that does not carry it, so the host gets back less than it validated", field)
		}
	}

	// The values, not just the keys: a decode that kept the key and lost the value would
	// pass a keys-only assertion.
	if got["package_manager"] != "dnf" {
		t.Errorf("package_manager = %v, want \"dnf\"", got["package_manager"])
	}
	if got["family"] != "fedora" {
		t.Errorf("family = %v, want \"fedora\"", got["family"])
	}
}

// TestInvokeLoadStillServesDistro is the other half: adding a word must not stop the FIRST
// word working. A `distro:` body carries none of the five os-only fields, so it must decode
// and round-trip without them rather than erroring on a required-field rule that belongs to
// the other word.
func TestInvokeLoadStillServesDistro(t *testing.T) {
	// A distro body carries NONE of the five os-only fields. `inherits` is a distro-only
	// build-vocabulary field, so it proves this path really decoded into spec.Distro.
	const distroBody = `{"inherits": "fedora", "version": "43"}`

	reply, err := provider{}.Invoke(context.Background(), &pb.InvokeRequest{
		Op:         sdk.OpLoad,
		Reserved:   "distro",
		ParamsJson: []byte(distroBody),
	})
	if err != nil {
		t.Fatalf("Invoke(OpLoad, distro): %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(reply.GetResultJson(), &got); err != nil {
		t.Fatalf("decoding the reply: %v", err)
	}
	if got["inherits"] != "fedora" {
		t.Errorf("inherits = %v, want \"fedora\" — the distro word stopped round-tripping", got["inherits"])
	}
	// The os-only fields must NOT appear: if this path had fallen into the os arm, an empty
	// description/family would be marshalled back and the host would see a distro that
	// claims to be an os.
	for _, field := range []string{"description", "family", "package_manager", "init"} {
		if _, ok := got[field]; ok {
			t.Errorf("OpLoad for word \"distro\" returned %q — that is an os field, so the "+
				"request was routed into the wrong arm", field)
		}
	}
}
