package distrokind

// resolve.go — candy/plugin-distro's OpResolve leg (the distro de-type, Cutover M):
// project an authored spec.Distro into a ResolvedDistro the kernel's build engine
// consumes without importing the concrete kind. Field-copy: the host keeps
// RenderTemplate + the cache-mount vocab; the plugin owns the distro knowledge.
//
// A HAND-WRITTEN field copy is a silent-drop hazard, and it has already dropped twice:
// disk_layout and installer were added to spec's #Distro AND to this plugin's own
// #DistroInput, so authoring them was accepted everywhere and validated cleanly — and
// then they were never carried across this function, so every consumer saw nil. See
// resolve_parity_test.go, which now fails on any spec field this copy forgets.

import (
	"encoding/json"
	"fmt"

	"github.com/opencharly/spec/spec"
)

func resolveDistro(in spec.DistroResolveInput) (spec.DistroResolveReply, error) {
	var d spec.Distro
	if err := json.Unmarshal(in.Distro, &d); err != nil {
		return spec.DistroResolveReply{}, fmt.Errorf("distro resolve: decode: %w", err)
	}
	return spec.DistroResolveReply{Resolved: &spec.ResolvedDistro{
		Inherits:        d.Inherits,
		InheritPackages: d.InheritPackages,
		Version:         d.Version,
		Bootstrap:       d.Bootstrap,
		Workarounds:     d.Workarounds,
		Format:          d.Format,
		BaseUser:        d.BaseUser,
		Pacstrap:        d.Pacstrap,
		Debootstrap:     d.Debootstrap,
		AlpineBootstrap: d.AlpineBootstrap,
		Bootloader:      d.Bootloader,
		DiskLayout:      d.DiskLayout,
		Installer:       d.Installer,
		Dnf:             d.Dnf,
		Raw:             in.Distro,
	}}, nil
}
