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
	"strings"
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

// TestInvokeResolveRefusesTheOsWord is the other half of serving two words.
//
// NewMeta advertises `os` to a host that dispatches kind-blindly, so an `os` request CAN
// reach OpResolve — and that arm decodes into spec.DistroResolveInput, which carries none of
// the five fields that make a body an os. Resolving it silently would build a ResolvedDistro
// from a truncated body: the defect the OpLoad arm exists to prevent, one operation over.
//
// Refusing is the honest answer while os resolution is unimplemented, and the assertion is
// specifically that it is LOUD rather than that it returns a particular message — a future
// os arm will replace this error and this test should be rewritten with it, not deleted.
func TestInvokeResolveRefusesTheOsWord(t *testing.T) {
	_, err := provider{}.Invoke(context.Background(), &pb.InvokeRequest{
		Op:         sdk.OpResolve,
		Reserved:   "os",
		ParamsJson: []byte(osBody),
	})
	if err == nil {
		t.Fatal("OpResolve accepted the \"os\" word — it decodes into spec.DistroResolveInput, " +
			"which drops the five os-only fields, so a build would receive a truncated body")
	}
	if !strings.Contains(err.Error(), `"os"`) {
		t.Errorf("the refusal does not name the word it refused: %v", err)
	}
}

// sourceBody is a complete authored `source:` node — the `oci` origin, which is the one that
// needs no download and no builder, plus the two fields every origin carries.
const sourceBody = `{
	"oci": "quay.io/archlinux/archlinux:base-20260719.0.558306",
	"description": "Arch base image",
	"os": "arch"
}`

// TestInvokeLoadKeepsSourceOrigins proves the source arm round-trips an authored body.
//
// spec.Source is `map[string]any` because the contract's #Source is an S4 union and a union
// has no single Go struct. That is safe HERE — a map keeps every key, so no origin's fields
// can be dropped — but it is only safe if the arm actually routes the word to that type
// instead of falling through to spec.Distro, which knows nothing of `oci`/`disk`/`iso`.
func TestInvokeLoadKeepsSourceOrigins(t *testing.T) {
	reply, err := provider{}.Invoke(context.Background(), &pb.InvokeRequest{
		Op:         sdk.OpLoad,
		Reserved:   "source",
		ParamsJson: []byte(sourceBody),
	})
	if err != nil {
		t.Fatalf("Invoke(OpLoad, source): %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(reply.GetResultJson(), &got); err != nil {
		t.Fatalf("decoding the reply: %v", err)
	}
	for _, field := range []string{"oci", "description", "os"} {
		if _, ok := got[field]; !ok {
			t.Errorf("OpLoad for word \"source\" dropped %q — the body was routed to a type that "+
				"does not carry it", field)
		}
	}
	if got["os"] != "arch" {
		t.Errorf("os = %v, want \"arch\" — D-SRC-2 makes it the field the origin is resolved against",
			got["os"])
	}
	// No distro-only field may appear: that is what catches a request routed into the wrong arm.
	if _, ok := got["inherits"]; ok {
		t.Error("OpLoad for word \"source\" returned `inherits`, a distro field — the request " +
			"was routed into the wrong arm")
	}
}

// TestInvokeLoadSourceKeepsAnUnmodelledOriginKey is the property a map buys, asserted.
//
// The gate (#SourceInput) rejects a malformed source before this arm sees it, so this body is
// NOT something an author could land — the test exists to show that the CANONICALISER cannot
// itself drop an origin's payload, which is the failure mode that bit the `os` word when a
// narrower struct was used.
func TestInvokeLoadSourceKeepsAnUnmodelledOriginKey(t *testing.T) {
	body := `{"disk": {"url": "https://example.invalid/a.qcow2", "sha256": "` + repeatHex64s() + `"}, "description": "d", "os": "arch"}`
	reply, err := provider{}.Invoke(context.Background(), &pb.InvokeRequest{
		Op:         sdk.OpLoad,
		Reserved:   "source",
		ParamsJson: []byte(body),
	})
	if err != nil {
		t.Fatalf("Invoke(OpLoad, source): %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(reply.GetResultJson(), &got); err != nil {
		t.Fatalf("decoding the reply: %v", err)
	}
	disk, ok := got["disk"].(map[string]any)
	if !ok {
		t.Fatalf("the disk origin did not survive canonicalisation: %v", got["disk"])
	}
	if disk["sha256"] == nil {
		t.Error("the disk origin lost its sha256 — D-SRC-1 pins every remote origin, so a " +
			"canonicaliser that drops it makes an unpinned source representable")
	}
}

// TestInvokeResolveRefusesTheSourceWord — OpResolve resolves `distro:` bodies only, and a
// source body decoded into spec.DistroResolveInput would be truncated exactly as an os body
// would. The guard is one branch for every non-distro word, so this asserts the branch covers
// the NEW word rather than assuming it.
func TestInvokeResolveRefusesTheSourceWord(t *testing.T) {
	_, err := provider{}.Invoke(context.Background(), &pb.InvokeRequest{
		Op:         sdk.OpResolve,
		Reserved:   "source",
		ParamsJson: []byte(sourceBody),
	})
	if err == nil {
		t.Fatal("OpResolve accepted the \"source\" word — it decodes into spec.DistroResolveInput, " +
			"which knows nothing of oci/disk/iso, so a build would receive a truncated body")
	}
	if !strings.Contains(err.Error(), `"source"`) {
		t.Errorf("the refusal does not name the word it refused: %v", err)
	}
}

// repeatHex64s is a sha256-shaped literal for fixtures.
func repeatHex64s() string { return strings.Repeat("a", 64) }
