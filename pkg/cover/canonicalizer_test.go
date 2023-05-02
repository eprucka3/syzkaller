// Copyright 2023 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

// Tests the translation of coverage pcs between fuzzer instances with differing module offsets.

package cover_test

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/google/syzkaller/pkg/cover"
	"github.com/google/syzkaller/pkg/host"
	"github.com/google/syzkaller/pkg/signal"
)

type RPCServer struct {
	canonicalModules   *cover.Canonicalizer
	modulesInitialized bool
	fuzzers            map[string]*Fuzzer
}

type Fuzzer struct {
	instModules *cover.CanonicalizerInstance
}

// Confirms there is no change to coverage if modules aren't instantiated.
func TestNilModules(t *testing.T) {
	serv := &RPCServer{
		fuzzers: make(map[string]*Fuzzer),
	}
	serv.Connect("f1", nil)
	serv.Connect("f2", nil)

	pcs := []uint32{0x00010011, 0x00020FFF, 0x00030000, 0x00040000}
	testCov := pcs
	goalOut := pcs

	// Concatenate PCs with a hash to reflect executor signal creation.
	signalsRaw := []uint32{0x000, 0xAAA, 0xBBB, 0xCCC}
	for idx := range signalsRaw {
		signalsRaw[idx] |= pcs[idx] & 0xFFFFF000
	}
	signals := signal.FromRaw(signalsRaw, 0)
	testSignals := signals.Serialize()

	for name, fuzzer := range serv.fuzzers {
		fuzzer.instModules.Canonicalize(testCov, testSignals)
		for idx, cov := range testCov {
			if cov != goalOut[idx] {
				failMsg := fmt.Errorf("fuzzer %v.\nExpected: 0x%x.\nReturned: 0x%x",
					name, goalOut[idx], cov)
				t.Fatalf("failed in pc canonicalization. %v", failMsg)
			}
		}
		if !reflect.DeepEqual(testSignals.Deserialize(), signals) {
			failMsg := fmt.Errorf("fuzzer %v.\nExpected: 0x%x.\nReturned: 0x%x",
				name, signals, testSignals.Deserialize())
			t.Fatalf("failed in signal canonicalization. %v", failMsg)
		}

		fuzzer.instModules.Decanonicalize(testCov, testSignals)
		for idx, cov := range testCov {
			if cov != goalOut[idx] {
				failMsg := fmt.Errorf("fuzzer %v.\nExpected: 0x%x.\nReturned: 0x%x",
					name, pcs[idx], cov)
				t.Fatalf("failed in pc decanonicalization. %v", failMsg)
			}
		}
		if !reflect.DeepEqual(testSignals.Deserialize(), signals) {
			failMsg := fmt.Errorf("fuzzer %v.\nExpected: 0x%x.\nReturned: 0x%x",
				name, signals, testSignals.Deserialize())
			t.Fatalf("failed in signal decanonicalization. %v", failMsg)
		}
	}
}

// Tests coverage conversion when modules are instantiated.
func TestModules(t *testing.T) {
	serv := &RPCServer{
		fuzzers: make(map[string]*Fuzzer),
	}

	// Create modules at the specified address offsets.
	var f1Modules, f2Modules []host.KernelModule
	f1ModuleAddresses := []uint64{0x00015000, 0x00020000, 0x00030000, 0x00040000, 0x00045000}
	f1ModuleSizes := []uint64{0x5000, 0x5000, 0x10000, 0x5000, 0x10000}

	f2ModuleAddresses := []uint64{0x00015000, 0x00040000, 0x00045000, 0x00020000, 0x00030000}
	f2ModuleSizes := []uint64{0x5000, 0x5000, 0x10000, 0x5000, 0x10000}
	for idx, address := range f1ModuleAddresses {
		f1Modules = append(f1Modules, host.KernelModule{
			Name: strconv.FormatInt(int64(idx), 10),
			Addr: address,
			Size: f1ModuleSizes[idx],
		})
	}
	for idx, address := range f2ModuleAddresses {
		f2Modules = append(f2Modules, host.KernelModule{
			Name: strconv.FormatInt(int64(idx), 10),
			Addr: address,
			Size: f2ModuleSizes[idx],
		})
	}

	serv.Connect("f1", f1Modules)
	serv.Connect("f2", f2Modules)

	testCov := make(map[string][]uint32)
	covOut := make(map[string][]uint32)

	pcs := []uint32{0x00010011, 0x00015F00, 0x00020FFF, 0x00025000, 0x00030000,
		0x00035000, 0x00040000, 0x00045000, 0x00050000, 0x00055000}

	// f1 is the "canonical" fuzzer as it is first one instantiated.
	// This means that all coverage output should be the same as the inputs.
	testCov["f1"] = pcs
	covOut["f1"] = pcs

	// The modules addresss are inverted between: (2 and 4), (3 and 5),
	// affecting the output canonical coverage values in these ranges.
	testCov["f2"] = pcs
	covOut["f2"] = []uint32{0x00010011, 0x00015F00, 0x00040FFF, 0x00025000, 0x00045000,
		0x0004a000, 0x00020000, 0x00030000, 0x0003b000, 0x00055000}

	// Concatenate PCs with a hash to reflect executor signal creation.
	signalsRaw := []uint32{0x000, 0xAAA, 0xBBB, 0xCCC, 0xDDD, 0xFFF, 0x111, 0x222, 0x333, 0x444}
	for idx := range signalsRaw {
		signalsRaw[idx] |= pcs[idx] & 0xFFFFF000
	}

	signals := signal.FromRaw(signalsRaw, 0)
	testSignals := make(map[string]signal.Serial)
	signalsOut := make(map[string]signal.Signal)

	testSignals["f1"] = signals.Serialize()
	signalsOut["f1"] = signals

	// The module addresses should be inverted similarly to coverage,
	// but the hash should be unaffected.
	testSignals["f2"] = signals.Serialize()
	signalsOut["f2"] = signal.FromRaw([]uint32{0x00010000, 0x00015AAA, 0x00040BBB, 0x00025CCC, 0x00045DDD,
		0x0004aFFF, 0x00020111, 0x00030222, 0x0003b333, 0x00055444}, 0)

	for name, fuzzer := range serv.fuzzers {
		// Test address conversion from instance to canonical.
		fuzzer.instModules.Canonicalize(testCov[name], testSignals[name])
		for idx, cov := range testCov[name] {
			if cov != covOut[name][idx] {
				failMsg := fmt.Errorf("fuzzer %v.\nExpected: 0x%x.\nReturned: 0x%x",
					name, covOut[name][idx], cov)
				t.Fatalf("failed in pc canonicalization. %v", failMsg)
			}
		}
		if !reflect.DeepEqual(testSignals[name].Deserialize(), signalsOut[name]) {
			failMsg := fmt.Errorf("fuzzer %v.\nExpected: 0x%x.\nReturned: 0x%x",
				name, signalsOut[name], testSignals[name].Deserialize())
			t.Fatalf("failed in signal canonicalization. %v", failMsg)
		}

		// Test address conversion from canonical to instance.
		fuzzer.instModules.Decanonicalize(testCov[name], testSignals[name])
		for idx, cov := range testCov[name] {
			if cov != pcs[idx] {
				failMsg := fmt.Errorf("fuzzer %v.\nExpected: 0x%x.\nReturned: 0x%x",
					name, pcs[idx], cov)
				t.Fatalf("failed in pc decanonicalization. %v", failMsg)
			}
		}
		if !reflect.DeepEqual(testSignals[name].Deserialize(), signals) {
			failMsg := fmt.Errorf("fuzzer %v.\nExpected: 0x%x.\nReturned: 0x%x",
				name, signals, testSignals[name].Deserialize())
			t.Fatalf("failed in signal decanonicalization. %v", failMsg)
		}
	}
}

func (serv *RPCServer) Connect(name string, modules []host.KernelModule) {
	if !serv.modulesInitialized {
		serv.canonicalModules = cover.NewCanonicalizer(modules)
		serv.modulesInitialized = true
	}

	serv.fuzzers[name] = &Fuzzer{
		instModules: serv.canonicalModules.NewInstance(modules),
	}
}
