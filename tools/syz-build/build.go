// Copyright 2021 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

// syz-build is a wrapper around pkg/build for testing purposes.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/google/syzkaller/pkg/build"
	"github.com/google/syzkaller/pkg/debugtracer"
	"github.com/google/syzkaller/pkg/mgrconfig"
	"github.com/google/syzkaller/pkg/tool"
)

var (
	flagCompiler      = flag.String("compiler", "", "non-defult compiler")
	flagLinker        = flag.String("linker", "", "non-default linker")
	flagKernelConfig  = flag.String("config", "", "kernel config file")
	flagKernelSysctl  = flag.String("sysctl", "", "kernel sysctl file")
	flagKernelCmdline = flag.String("cmdline", "", "kernel cmdline file")
	flagUserspace     = flag.String("userspace", "", "path to userspace for build")
	flagKernelSrc     = flag.String("kernel_src", "", "path to kernel checkout")
	flagMgrConfig     = flag.String("mgrconfig", "", "manager config")
	flagTrace         = flag.Bool("trace", false, "trace build process and save debug artefacts")
)

func main() {
	flag.Parse()
	if os.Getuid() != 0 {
		fmt.Printf("not running under root, image build may fail\n")
	}
	os.Setenv("SYZ_DISABLE_SANDBOXING", "yes")
	kernelConfig := []byte{}
	var err error
	if *flagKernelConfig != "" {
		kernelConfig, err = os.ReadFile(*flagKernelConfig)
		if err != nil {
			tool.Fail(err)
		}
	}

	cfg, err := mgrconfig.LoadPartialFile(*flagMgrConfig)
	if err != nil {
		tool.Fail(err)
	}

	params := build.Params{
		TargetOS:     cfg.TargetOS,
		TargetArch:   cfg.TargetVMArch,
		VMType:       cfg.Type,
		KernelDir:    *flagKernelSrc,
		OutputDir:    ".",
		Compiler:     *flagCompiler,
		Linker:       *flagLinker,
		Ccache:       "",
		UserspaceDir: *flagUserspace,
		CmdlineFile:  *flagKernelCmdline,
		SysctlFile:   *flagKernelSysctl,
		Config:       kernelConfig,
		Tracer:       &debugtracer.NullTracer{},
		Build:        cfg.Build,
	}
	if *flagTrace {
		params.Tracer = &debugtracer.GenericTracer{
			TraceWriter: os.Stdout,
			OutDir:      ".",
		}
	}
	details, err := build.Image(params)
	if err != nil {
		tool.Fail(err)
	}
	params.Tracer.Log("signature: %v", details.Signature)
	params.Tracer.Log("compiler: %v", details.CompilerID)
}
