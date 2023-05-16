// Copyright 2023 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package cover

import (
	"sort"

	"github.com/google/syzkaller/pkg/host"
	"github.com/google/syzkaller/pkg/signal"
)

type Canonicalizer struct {
	// Map of modules stored as module name:kernel offset.
	modules map[string]uint32

	// Contains a sorted list of the canonical module addresses.
	moduleKeys []uint32
}

type CanonicalizerInstance struct {
	canonical Canonicalizer

	// Contains a sorted list of the instance's module addresses.
	moduleKeys []uint32

	// Contains a map of the uint32 address to the necessary offset.
	instToCanonicalMap map[uint32]*canonicalizerModule
	canonicalToInstMap map[uint32]*canonicalizerModule
}

// Contains the offset and final address of each module.
type canonicalizerModule struct {
	offset  int
	endAddr uint32
}

func NewCanonicalizer(modules []host.KernelModule) *Canonicalizer {
	// Create a map of canonical module offsets by name.
	canonicalModules := make(map[string]uint32)
	for _, module := range modules {
		canonicalModules[module.Name] = uint32(module.Addr)
	}

	// Store sorted canonical address keys.
	canonicalModuleKeys := make([]uint32, len(modules))
	setModuleKeys(canonicalModuleKeys, modules)
	return &Canonicalizer{
		modules:    canonicalModules,
		moduleKeys: canonicalModuleKeys,
	}
}

func (can *Canonicalizer) NewInstance(modules []host.KernelModule) *CanonicalizerInstance {
	// Save sorted list of module offsets.
	moduleKeys := make([]uint32, len(modules))
	setModuleKeys(moduleKeys, modules)

	// Create a hash between the "canonical" module addresses and each VM instance.
	instToCanonicalMap := make(map[uint32]*canonicalizerModule)
	canonicalToInstMap := make(map[uint32]*canonicalizerModule)
	for _, module := range modules {
		canonicalAddr := can.modules[module.Name]
		instAddr := uint32(module.Addr)

		canonicalModule := &canonicalizerModule{
			offset:  int(instAddr) - int(canonicalAddr),
			endAddr: uint32(module.Size) + canonicalAddr,
		}
		canonicalToInstMap[canonicalAddr] = canonicalModule

		instModule := &canonicalizerModule{
			offset:  int(canonicalAddr) - int(instAddr),
			endAddr: uint32(module.Size) + instAddr,
		}
		instToCanonicalMap[instAddr] = instModule
	}

	return &CanonicalizerInstance{
		canonical:          *can,
		moduleKeys:         moduleKeys,
		instToCanonicalMap: instToCanonicalMap,
		canonicalToInstMap: canonicalToInstMap,
	}
}

func (ci *CanonicalizerInstance) Canonicalize(cov []uint32, sign signal.Serial) {
	// Skip conversion if modules are not used.
	if len(ci.moduleKeys) == 0 {
		return
	}
	convertModulePCs(ci.moduleKeys, ci.instToCanonicalMap, cov)
	convertSignals(ci.moduleKeys, ci.instToCanonicalMap, sign)
}

func (ci *CanonicalizerInstance) Decanonicalize(cov []uint32, sign signal.Serial) {
	// Skip conversion if modules are not used.
	if len(ci.canonical.moduleKeys) == 0 {
		return
	}
	convertModulePCs(ci.canonical.moduleKeys, ci.canonicalToInstMap, cov)
	convertSignals(ci.canonical.moduleKeys, ci.canonicalToInstMap, sign)
}

func (ci *CanonicalizerInstance) DecanonicalizeFilter(pcs map[uint32]uint32) map[uint32]uint32 {
	// Skip conversion if modules or filtering are not used.
	if len(ci.canonical.moduleKeys) == 0 || len(pcs) == 0 {
		return pcs
	}

	// Deserialize PCs for conversion.
	cov := make([]uint32, len(pcs))
	vals := make([]uint32, len(pcs))
	idx := 0
	for pc, val := range pcs {
		cov[idx] = pc
		vals[idx] = val
		idx++
	}
	convertModulePCs(ci.canonical.moduleKeys, ci.canonicalToInstMap, cov)

	// Recreate cover filter map.
	instPCs := make(map[uint32]uint32)
	for idx, pc := range cov {
		instPCs[pc] = vals[idx]
	}
	return instPCs

}

// Store sorted list of addresses. Used to binary search when converting PCs.
func setModuleKeys(moduleKeys []uint32, modules []host.KernelModule) {
	for idx, module := range modules {
		// Truncate PCs to uint32, assuming that they fit into 32 bits.
		// True for x86_64 and arm64 without KASLR.
		moduleKeys[idx] = uint32(module.Addr)
	}

	// Sort modules by address.
	sort.Slice(moduleKeys, func(i, j int) bool { return moduleKeys[i] < moduleKeys[j] })
}

func findModule(pc uint32, moduleKeys []uint32) (moduleIdx int) {
	moduleIdx, _ = sort.Find(len(moduleKeys), func(moduleIdx int) int {
		if pc < moduleKeys[moduleIdx] {
			return -1
		}
		return +1
	})
	// Sort.Find returns the index above the correct module.
	return moduleIdx - 1
}

func convertModulePCs(moduleKeys []uint32, conversionHash map[uint32]*canonicalizerModule, cov []uint32) {
	for idx, pc := range cov {
		moduleIdx := findModule(pc, moduleKeys)
		// Check if address is above the first module offset.
		if moduleIdx >= 0 {
			module := conversionHash[moduleKeys[moduleIdx]]
			// If the address is within the found module add the offset.
			if pc < module.endAddr {
				cov[idx] = uint32(int(pc) + module.offset)
			}
		}
	}
}

func convertSignals(moduleKeys []uint32, conversionHash map[uint32]*canonicalizerModule, sign signal.Serial) {
	for idx, elem := range sign.Elems {
		moduleIdx := findModule(uint32(elem), moduleKeys)
		// Check if address is above the first module offset.
		if moduleIdx >= 0 {
			module := conversionHash[moduleKeys[moduleIdx]]
			// If the address is within the found module add the offset.
			if uint32(elem) < module.endAddr {
				sign.UpdateElem(idx, uint32(int(elem)+module.offset))
			}
		}
	}
}
