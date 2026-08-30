package distrokind

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// This plugin serves `kind: distro`, and it validates an authored `distro:` body against
// its OWN self-contained #DistroInput before dispatching. That duplication is sanctioned —
// the plugin schema must compile standalone while spec's #Distro must generate Go — but it
// means the two can DRIFT, and when they do the failure is confusing: charly's core
// accepts a field and this plugin rejects it with `#DistroInput.<field>: field not allowed`,
// which reads like the field does not exist at all.
//
// That is not hypothetical. spec gained `#Distro.installer` and this def did not, so an
// authored installer: block was rejected here for months. `disk_layout` was about to
// repeat it. This test makes the next omission fail HERE, next to the fix, instead of in
// a consumer's build log.
//
// It asserts the TOP-LEVEL field set only. The inner defs deliberately differ in name
// (#DsBootloader vs #Bootloader) and in Go annotations, so comparing them would be noise;
// what matters is that every field a distro entity can author is authorable here.
func TestDistroInputCoversEverySpecDistroField(t *testing.T) {
	local, err := os.ReadFile(filepath.Join("schema", "distro.cue"))
	if err != nil {
		t.Fatalf("reading this plugin's schema: %v", err)
	}
	got := topLevelFields(t, string(local), "#DistroInput")

	// The field set spec's #Distro authorises, as of the spec this plugin is written
	// against. Kept as a literal rather than read from the spec module because the
	// plugin's schema is deliberately standalone — importing spec here to check
	// standalone-ness would defeat the point.
	want := []string{
		"alpine_bootstrap", "base_user", "bootloader", "bootstrap", "debootstrap",
		"disk_layout", "dnf", "format", "inherit_packages", "inherits", "installer",
		"pacstrap", "version", "workaround",
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("#DistroInput fields drifted from spec's #Distro.\n got: %v\nwant: %v\n"+
			"If spec added a field, mirror it here; if spec removed one, remove it here "+
			"and from `want`.", got, want)
	}
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
