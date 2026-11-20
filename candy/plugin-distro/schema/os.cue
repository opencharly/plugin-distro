// os.cue — the `os` KIND's gate for plugin-distro.
//
// WHY THIS DEF MIRRORS spec's #Os AND NOT #DistroInput: the two spell three concepts
// differently — `inherit`/`inherit_package` against `inherits`/`inherit_packages`, and a
// grouped `bootstrapper` against three flattened fields. An `os:` gate that mirrored the
// legacy names would accept bodies core then rejects, which is the drift this file's
// sibling parity test exists to catch; it caught exactly these three on the first run.
//
// WHY THE DUPLICATION IS SANCTIONED: the plugin's
// duplication of spec's authored surface is SANCTIONED (see schema_parity_test.go's
// header — the plugin schema must compile standalone while spec's defs must also generate
// Go), and #DistroInput is a CLOSED struct, so it cannot be unified with five extra
// fields. What is reused rather than duplicated is SIX of the ten inner shapes
// (#DsBootstrap, #DsFormat, #DsDiskLayout, #DsBaseUser, #DsBootloader, #DsInstaller) — the
// same defs #DistroInput uses, so an os node and a distro node cannot disagree about what a
// `format:` block is. The other four are described under THE PARITY GUARD below.
//
// What an `os:` node adds to a distro node is the five fields that make it an OPERATING
// SYSTEM rather than a set of build hints: who it is (description, family), what installs
// into it (package_manager), what starts its services (init), and how its disk form boots
// (boot). The four required ones are D-CANDY-3 and D-SRC-2's — the os owns the init, and a
// source names the os precisely so that os owns every form conversion. An os node that
// omitted them could not answer the one question #Source asks it.
//
// THE PARITY GUARD: TestOsInputAddsExactlyOsFields, in schema_parity_test.go, compares this
// def's TOP-LEVEL field names against spec's #Os (added in opencharly/spec#402) in both
// directions, for the reason that file's header gives: when the two drift, the failure reads
// as "the field does not exist" rather than "the two defs disagree". It compares NAMES only
// and does not descend into the inner defs — and those are TWO kinds. Six are #Ds* defs
// shared with #DistroInput, already guarded by TestEveryMirroredDefMatchesSpec. Four are NOT:
// #OsPackageManager, #OsInit, #OsBoot and #OsBootstrapper are this file's own names, so that
// sweep never reaches them, and TestOsOnlyInnerDefsMatchSpec guards exactly those four
// against spec's _packageManager / _initRole / #Boot / #OsBootstrapper.

// ── the five fields an os adds ───────────────────────────────────────────────

// #OsPackageManager — the package managers charly drives (DESIGN.cue:166).
#OsPackageManager: *"dnf" | "apt" | "pacman" | "apk" | "zypper"

// #OsInit — the init node per role (DESIGN.cue:836). Both roles are REQUIRED: an os that
// named only one could not build both a container and a system image of itself.
#OsInit: {
	container!: string & !=""
	system!:    string & !=""
}

// #OsBoot — the boot content of the disk form (DESIGN.cue:839).
#OsBoot: {
	firmware: *"uefi" | "bios" | "uefi_secure"
	kernel?:  string & !=""
	initrd?:  string & !=""
	command_line?: string & !=""
}

// #OsInput — the authored body of an `os:` node.
//
// The shared half names the SAME inner defs as #DistroInput, so a `bootstrap:` or a
// `format:` block means the same thing on both kinds; that is what keeps the two from
// becoming two vocabularies while still letting each def compile standalone.
#OsInput: {
	description!:     string & !=""
	family!:          string & =~"^[a-z0-9]+([.-][a-z0-9]+)*$"
	version?:         string & !=""
	package_manager!: #OsPackageManager
	init!:            #OsInit
	boot?:            #OsBoot

	// §16.4: the os of the stack's source owns every CONVERSION.
	inherit?:         [...string]
	inherit_package?: [...string]
	bootstrap?:       #DsBootstrap
	workaround?:      [...string]
	format?:          {[string]: #DsFormat}
	base_user?:       #DsBaseUser
	bootstrapper?:    #OsBootstrapper
	bootloader?:      #DsBootloader
	disk_layout?:     #DsDiskLayout
	installer?:       #DsInstaller
}

// #OsBootstrapper — the from-scratch tools, by the family that ships them. The contract
// groups the three the build vocabulary flattens (distro.cue's pacstrap / debootstrap /
// alpine_bootstrap), which is why this def exists instead of three fields: an `os:` node
// says WHICH bootstrappers it has, and the grouping is what makes "an os may have several"
// expressible as one field rather than as an implicit convention.
#OsBootstrapper: {
	pacstrap?:    string & !=""
	debootstrap?: string & !=""
	alpine?:      string & !=""
}
