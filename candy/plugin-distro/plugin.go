// Package distrokind is the importable form of charly's `distro` plugin KIND. A KIND provider
// dispatches via the pb Invoke(OpLoad) envelope — decode the authored `distro:` entity into
// the core spec.Distro and re-marshal as canonical JSON; the host lands it in
// uf.PluginKinds["distro"][<name>]. Usable COMPILED-IN (NewProvider()/NewMeta() via
// plugins_generated.go) OR served OUT-OF-PROCESS by the cmd/serve shim. Relocated out of
// charly's module (formerly charly/plugin/builtins/distro + charly/plugin_distro.go).
package distrokind

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"

	"github.com/opencharly/sdk"
	pb "github.com/opencharly/spec/proto"
	"github.com/opencharly/spec/spec"
)

//go:embed schema/*.cue
var schemaFS embed.FS

// NewProvider returns the kind provider for in-proc registration or out-of-proc serving.
func NewProvider() pb.ProviderServer { return &provider{} }

// NewMeta ships this plugin's TWO kind capabilities + its self-contained CUE schema, via
// sdk.NewMeta → BuildCapabilities.
//
// TWO WORDS, ONE VOCABULARY. `distro:` is the build vocabulary a box declares; `os:` is the
// OPERATING SYSTEM a source names (DESIGN.cue:284, D-SRC-2). They share every inner shape —
// a `bootstrap:` or `format:` block means the same thing on both — and differ only in what
// an os must say about itself: description, family, package_manager, init. Serving both from
// one plugin is what keeps the two from becoming two vocabularies, which is the R2 debt
// opencharly/opencharly#1200 records.
//
// The word is not implied by the plugin: a multi-word provider learns it from
// req.GetReserved() (see Invoke), which is why serving a second kind needs a case there and
// not only a row here.
func NewMeta() pb.PluginMetaServer {
	return sdk.NewMeta("2026.176.3203",
		[]sdk.ProvidedCapability{
			{Class: "kind", Word: "distro", InputDef: "#DistroInput"},
			{Class: "kind", Word: "os", InputDef: "#OsInput"},
			{Class: "kind", Word: "source", InputDef: "#SourceInput"},
		},
		schemaFS)
}

type provider struct{ pb.UnimplementedProviderServer }

// Invoke handles OpLoad: decode the authored `distro:` entity into spec.Distro and return it
// re-marshalled as canonical JSON (the host validated the body against #DistroInput first).
func (provider) Invoke(_ context.Context, req *pb.InvokeRequest) (*pb.InvokeReply, error) {
	switch req.GetOp() {
	case sdk.OpLoad:
		// The host dispatches kind-blindly, so the WORD arrives on the request rather than
		// being implied by the plugin. Two words, two canonical types: an os body carries the
		// five fields a distro body does not, so unmarshalling it into spec.Distro would
		// silently drop them and hand the host back a body it had just validated as complete.
		switch req.GetReserved() {
		case "os":
			return opLoadCanonical[spec.Os]("os", req)
		case "source":
			// spec.Source is `map[string]any` — the contract's #Source is an S4 union, and a
			// union has no single Go struct. That is FINE here and is why this word is safe:
			// a map round-trips every key, so the canonical form cannot drop an origin's
			// fields the way decoding into a narrower struct would. The gate
			// (#SourceInput) is what rejects a malformed body, before this arm sees it.
			return opLoadCanonical[spec.Source]("source", req)
		}
		return opLoadCanonical[spec.Distro]("distro", req)
	case sdk.OpResolve:
		// The distro de-type (Cutover M): project the opaque distro body into a
		// ResolvedDistro the kernel's build engine consumes.
		//
		// WORD-AWARE, and it REFUSES rather than truncating. Now that NewMeta advertises the
		// `os` word, a host that dispatches kind-blindly can send an `os` request here too —
		// and this arm decodes into spec.DistroResolveInput, which carries none of the five
		// fields that make a body an os. Silently resolving it would produce a ResolvedDistro
		// built from a truncated body: the exact defect the OpLoad arm above exists to
		// prevent, one operation over.
		//
		// An error is the honest answer while `os` resolution is unimplemented: it makes a
		// misroute LOUD at the boundary instead of shipping a wrong resolved distro into a
		// build. When os resolution is designed, it gets its own arm here, not a fallthrough.
		if req.GetReserved() != "" && req.GetReserved() != "distro" {
			return nil, fmt.Errorf("distro kind: OpResolve is not implemented for word %q; "+
				"this arm resolves `distro:` bodies only, and decoding an %q body into "+
				"spec.DistroResolveInput would truncate it", req.GetReserved(), req.GetReserved())
		}
		var in spec.DistroResolveInput
		if len(req.GetParamsJson()) > 0 {
			if err := json.Unmarshal(req.GetParamsJson(), &in); err != nil {
				return nil, fmt.Errorf("distro resolve: decode input: %w", err)
			}
		}
		reply, err := resolveDistro(in)
		if err != nil {
			return nil, err
		}
		out, err := json.Marshal(reply)
		if err != nil {
			return nil, fmt.Errorf("distro resolve: marshal reply: %w", err)
		}
		return &pb.InvokeReply{ResultJson: out}, nil
	default:
		return nil, fmt.Errorf("distro kind: unsupported op %q", req.GetOp())
	}
}

// opLoadCanonical decodes an authored entity body into T and re-marshals it as canonical
// JSON, which is the whole of an OpLoad for a kind whose Go type is the authority on its own
// shape.
//
// GENERIC ON PURPOSE (R2): the `os` and `distro` arms differ ONLY by type, so writing them
// twice invites exactly the drift this file's parity tests exist to catch — one arm gaining a
// step the other lacks (a normalisation, a field fix-up) and the two words silently acquiring
// different canonical forms. The word is threaded through for the error prefix only, so a
// failure still names which kind it came from.
func opLoadCanonical[T any](word string, req *pb.InvokeRequest) (*pb.InvokeReply, error) {
	var in T
	if len(req.GetParamsJson()) > 0 {
		if err := json.Unmarshal(req.GetParamsJson(), &in); err != nil {
			return nil, fmt.Errorf("%s kind: decode entity: %w", word, err)
		}
	}
	out, err := json.Marshal(in)
	if err != nil {
		return nil, fmt.Errorf("%s kind: marshal entity: %w", word, err)
	}
	return &pb.InvokeReply{ResultJson: out}, nil
}
