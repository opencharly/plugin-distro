// source.cue — the `source` KIND's gate for plugin-distro.
//
// WHY IT LIVES HERE, beside `os`: DESIGN.cue:283-284 declares them as a PAIR — `source:` is the
// first of the ten kinds and `os:` the second — and the contract binds them: #SourceFields
// requires `os!: #Ref @ref(os)`. A source that does not name an os cannot say what converts it
// into the other form (§16.4), so the two vocabulary kinds are served by one plugin for the
// same reason `distro` and `os` are: two homes would make two vocabularies of one.
//
// WHAT A SOURCE IS (§14.1-14.2): an external starting point — one origin plus the os it
// provides. The origin is exactly one of four: `oci` (an image ref), `disk` (a pinned
// download), `iso` (a download plus the installer's answer file), `bootstrap` (a builder and
// the packages it wants).
//
// D-SRC-1 IS STRUCTURAL HERE, NOT ADVISORY. #SourceAlt is an S4 key-tagged union, so
// `or([...])` admits exactly ONE arm — two origins and zero origins are both unrepresentable.
// The `disk` and `iso` arms carry a #Download whose `sha256` is REQUIRED, so an unpinned
// remote origin cannot be written down at all.
//
// SELF-CONTAINED, like the rest of this directory: the builtin's schema is spliced onto the
// base as `base ++ plugin`, so a reference upward into another candy's CUE would not resolve.
// The four helper defs below mirror the contract's (#ImageRef at DESIGN.cue:67, #Download at
// :436, #SourceIso and #SourceBootstrap at :799-809); os_parity-style coverage for them lives
// in source_parity_test.go.

// #SourceImageRef — an OCI image reference, the way `podman pull` spells one: registry/repo,
// an optional tag, an optional digest. NOT #OciRef, which requires an explicit `oci://` scheme;
// a source's `oci:` origin carries none.
#SourceImageRef: =~"^[a-z0-9.-]+(:[0-9]+)?/[a-z0-9._/-]+(:[A-Za-z0-9._-]+)?(@sha256:[0-9a-f]{64})?$"

// #SourceDownload — a remote artefact, pinned. D-SRC-1: a remote origin is pinned by sha256,
// so an unpinned one is unrepresentable rather than merely discouraged.
#SourceDownload: {
	url!:    string & =~"^[a-z][a-z0-9+.-]*://[^ ]+$"
	sha256!: string & =~"^[0-9a-f]{64}$"
}

// #SourceIso — a disk image plus the installer's own answer file.
#SourceIso: {
	url!:    string & =~"^[a-z][a-z0-9+.-]*://[^ ]+$"
	sha256!: string & =~"^[0-9a-f]{64}$"
	config?: {...} // S10 — foreign-owned: the installer's own answer file, passed through
}

// #SourceBootstrap — a root filesystem made from scratch by a builder.
#SourceBootstrap: {
	builder!: string & !=""
	package?: {[string]: [...string]}
}

// #SourceInput — the authored body of a `source:` node.
//
// #SourceAlt is the union and #SourceInput the node: the node is the union plus the two fields
// EVERY origin carries (D-SRC-2's `os`, and the `description` that makes the node readable).
// They are spelled out per arm rather than spread from a helper, because this module's CUE
// carries no `@experiment(explicitopen)` and definition-spread needs it — and spelling them
// out is what makes "exactly one origin" readable where it is enforced.
#SourceAlt: {
	oci: {
		oci!:         #SourceImageRef
		description!: string & !=""
		os!:          string & !=""
	}
	disk: {
		disk!:        #SourceDownload
		description!: string & !=""
		os!:          string & !=""
	}
	iso: {
		iso!:         #SourceIso
		description!: string & !=""
		os!:          string & !=""
	}
	bootstrap: {
		bootstrap!:   #SourceBootstrap
		description!: string & !=""
		os!:          string & !=""
	}
}

// #SourceInput — one of the four origins, exactly one.
#SourceInput: or([
	#SourceAlt.oci,
	#SourceAlt.disk,
	#SourceAlt.iso,
	#SourceAlt.bootstrap,
])
