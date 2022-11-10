// Copyright 2022 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package build

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/syzkaller/pkg/log"
	"github.com/google/syzkaller/pkg/osutil"
	"github.com/google/syzkaller/sys/targets"
)

// ParamsConfig defines external module and build config paths from the input params.Config file.
type ParamsConfig struct {
	KernelConfig  string
	ModulesConfig string
	ExtModules    string
	ModulesScript string
}

type android struct{}

func (a android) readCompiler(archivePath string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	h, err := tr.Next()
	for ; err == nil; h, err = tr.Next() {
		if filepath.Base(h.Name) == "compile.h" {
			bytes, err := ioutil.ReadAll(tr)
			if err != nil {
				return "", err
			}
			result := linuxCompilerRegexp.FindSubmatch(bytes)
			if result == nil {
				return "", fmt.Errorf("include/generated/compile.h does not contain build information")
			}

			return string(result[1]), nil
		}
	}

	return "", fmt.Errorf("archive %s doesn't contain include/generated/compile.h", archivePath)
}

func (a android) buildKernel(configPath []byte, params Params) error {
	commonKernelDir := filepath.Join(params.KernelDir, "common")
	configFile := filepath.Join(commonKernelDir, ".config")
	if err := a.writeFile(configFile, configPath); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}
	// if err := a.runMake(commonKernelDir, params, "mrproper"); err != nil {
	// 	return err
	// }
	// log.Logf(0, "LIZ_TESTING: SLEEP")
	// time.Sleep(time.Hour * 8)
	// One would expect olddefconfig here, but olddefconfig is not present in v3.6 and below.
	// oldconfig is the same as olddefconfig if stdin is not set.
	if err := a.runMake(commonKernelDir, params, "oldconfig"); err != nil {
		return err
	}
	// Write updated kernel config early, so that it's captured on build failures.
	outputConfig := filepath.Join(params.OutputDir, "kernel.config")
	if err := osutil.CopyFile(configFile, outputConfig); err != nil {
		return err
	}
	// Ensure CONFIG_GCC_PLUGIN_RANDSTRUCT doesn't prevent ccache usage.
	// See /Documentation/kbuild/reproducible-builds.rst.
	const seed = `const char *randstruct_seed = "e9db0ca5181da2eedb76eba144df7aba4b7f9359040ee58409765f2bdc4cb3b8";`
	gccPluginsDir := filepath.Join(commonKernelDir, "scripts", "gcc-plugins")
	if osutil.IsExist(gccPluginsDir) {
		if err := a.writeFile(filepath.Join(gccPluginsDir, "randomize_layout_seed.h"), []byte(seed)); err != nil {
			return err
		}
	}

	// Different key is generated for each build if key is not provided.
	// see Documentation/reproducible-builds.rst. This is causing problems to our signature calculation.
	certsDir := filepath.Join(commonKernelDir, "certs")
	if osutil.IsExist(certsDir) {
		if err := a.writeFile(filepath.Join(certsDir, "signing_key.pem"), []byte(moduleSigningKey)); err != nil {
			return err
		}
	}
	// if err := a.runMake(commonKernelDir, params, "bzImage"); err != nil {
	// 	return err
	// }
	// log.Logf(0, "LIZ_TESTING: After make")
	// log.Logf(0, "LIZ_TESTING: SLEEP")
	// time.Sleep(time.Hour * 8)
	if err := a.runMake(commonKernelDir, params, "bzImage", "modules", "prepare-objtool"); err != nil {
		return err
	}

	moduleStagingDir := filepath.Join(commonKernelDir, "staging")
	moduleInstallFlag := fmt.Sprintf("INSTALL_MOD_PATH=%v", moduleStagingDir)
	if err := a.runMake(commonKernelDir, params, moduleInstallFlag, "modules_install"); err != nil {
		return err
	}
	return nil
}

func (a android) buildExtModules(extModulePath string, params Params) error {
	commonKernelDir := filepath.Join(params.KernelDir, "common")

	// Location of external modules relative to common kernel dir
	mFlag := fmt.Sprintf("M=../%v", extModulePath)
	// Absolute location of the kernel source directory
	srcFlag := fmt.Sprintf("KERNEL_SRC=%v", commonKernelDir)

	moduleStagingDir := filepath.Join(commonKernelDir, "staging")
	moduleInstallFlag := fmt.Sprintf("INSTALL_MOD_PATH=%v", moduleStagingDir)

	// Make external modules
	if err := a.runMake(params.KernelDir, params, "-C", extModulePath, mFlag, srcFlag, moduleInstallFlag); err != nil {
		return err
	}

	// Install modules
	if err := a.runMake(params.KernelDir, params, "-C", extModulePath, mFlag, srcFlag, moduleInstallFlag, "modules_install"); err != nil {
		return err
	}

	return nil
}

func (a android) build(params Params) (ImageDetails, error) {
	var details ImageDetails
	var err error

	if params.CmdlineFile != "" {
		return details, fmt.Errorf("cmdline file is not supported for android cuttlefish images")
	}
	if params.SysctlFile != "" {
		return details, fmt.Errorf("sysctl file is not supported for android cuttlefish images")
	}

	// Parse input config
	var paramsConfig ParamsConfig
	if err = json.Unmarshal(params.Config, &paramsConfig); err != nil {
		return details, fmt.Errorf("failed to unmarshal kernel config json: %v", err)
	}

	log.Logf(0, "LIZ_TESTING: kernelConfig: %v", paramsConfig.KernelConfig)
	log.Logf(0, "LIZ_TESTING: modulesConfig: %v", paramsConfig.ModulesConfig)
	// var kernelConfig, modulesConfig []byte
	// kernelConfig, err = ioutil.ReadFile(paramsConfig.KernelConfig)
	// if err != nil {
	// 	return details, fmt.Errorf("failed to read kernel config: %v", err)
	// }
	// modulesConfig, err = ioutil.ReadFile(paramsConfig.ModulesConfig)
	// if err != nil {
	// 	return details, fmt.Errorf("failed to read modules config: %v", err)
	// }

	commonKernelDir := filepath.Join(params.KernelDir, "common")
	// log.Logf(0, "LIZ_TESTING: commonKernelDir: %v", commonKernelDir)

	// Build common kernel
	// if err := a.buildKernel(kernelConfig, params); err != nil {
	// 	return details, fmt.Errorf("failed to build android common kernel: %v", err)
	// }
	// if err := osutil.CopyFile(filepath.Join(params.OutputDir, "kernel.config"), filepath.Join(params.OutputDir, "common-kernel.config")); err != nil {
	// 	return details, fmt.Errorf("failed to copy kernel config file: %v", err)
	// }

	// Build modules
	// if err := a.buildKernel(modulesConfig, params); err != nil {
	// 	return details, fmt.Errorf("failed to build android common modules: %v", err)
	// }
	// if err := osutil.CopyFile(filepath.Join(params.OutputDir, "kernel.config"), filepath.Join(params.OutputDir, "modules.config")); err != nil {
	// 	return details, fmt.Errorf("failed to copy modules config file: %v", err)
	// }

	// Build external modules
	if err := a.buildExtModules(paramsConfig.ExtModules, params); err != nil {
		return details, fmt.Errorf("failed to build external modules: %v", err)
	}

	// Zip kernel headers
	execModuleScript := fmt.Sprintf("./%v", paramsConfig.ModulesScript)
	if _, err := osutil.RunCmd(time.Hour, "", execModuleScript, "zip_kernel_headers", commonKernelDir, "common"); err != nil {
		return details, fmt.Errorf("failed to zip kernel headers: %v", err)
	}

	// Create initramfs image
	if _, err := osutil.RunCmd(time.Hour, "", execModuleScript, "create_initramfs", commonKernelDir); err != nil {
		return details, fmt.Errorf("failed to create initramfs image: %v", err)
	}

	bzImage := filepath.Join(commonKernelDir, "arch", "x86", "boot", "bzImage")
	vmlinux := filepath.Join(commonKernelDir, "vmlinux")
	initramfs := filepath.Join(commonKernelDir, "initramfs.img")

	details.CompilerID, err = a.readCompiler(filepath.Join(commonKernelDir, "kernel-headers.tar.gz"))
	if err != nil {
		return details, err
	}

	if err := embedFiles(params, func(mountDir string) error {
		homeDir := filepath.Join(mountDir, "root")

		if err := osutil.CopyFile(bzImage, filepath.Join(homeDir, "bzImage")); err != nil {
			return err
		}
		if err := osutil.CopyFile(vmlinux, filepath.Join(homeDir, "vmlinux")); err != nil {
			return err
		}
		if err := osutil.CopyFile(initramfs, filepath.Join(homeDir, "initramfs.img")); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return details, err
	}

	if err := osutil.CopyFile(vmlinux, filepath.Join(params.OutputDir, "obj", "vmlinux")); err != nil {
		return details, err
	}
	if err := osutil.CopyFile(initramfs, filepath.Join(params.OutputDir, "obj", "initrd")); err != nil {
		return details, err
	}

	details.Signature, err = elfBinarySignature(vmlinux, params.Tracer)
	if err != nil {
		return details, fmt.Errorf("failed to generate signature: %s", err)
	}

	return details, nil
}

func (a android) runMakeImpl(runDir, arch, compiler, linker, ccache, kernelDir string, extraArgs []string) error {
	target := targets.Get(targets.Linux, arch)
	args := LinuxMakeArgs(target, compiler, linker, ccache, "")
	args = append(args, extraArgs...)
	cmd := osutil.Command("make", args...)
	if err := osutil.Sandbox(cmd, true, true); err != nil {
		return err
	}
	cmd.Dir = runDir
	log.Logf(0, "LIZ_TESTING: dir: %v", cmd.Dir)
	log.Logf(0, "LIZ_TESTING: cmd: %v", cmd.Args)
	// Add prebuilts to path
	prebuilts := filepath.Join(kernelDir, "prebuilts/kernel-build-tools/linux-x86/bin/")
	env := os.Environ()
	for idx, envVar := range env {
		if strings.HasPrefix(envVar, "PATH=") {
			env[idx] = fmt.Sprintf("%v:%v", envVar, prebuilts)
		}
	}
	log.Logf(0, "LIZ_TESTING: env: %v", env)

	cmd.Env = append([]string{}, env...)
	// cmd.Env = append([]string{}, os.Environ()...)
	// log.Logf(0, "LIZ_TESTING: env: %v", cmd.Env)
	// log.Logf(0, "LIZ_TESTING: SLEEP")
	// time.Sleep(time.Hour * 8)
	// This makes the build [more] deterministic:
	// 2 builds from the same sources should result in the same vmlinux binary.
	// Build on a release commit and on the previous one should result in the same vmlinux too.
	// We use it for detecting no-op changes during bisection.
	cmd.Env = append(cmd.Env,
		"KBUILD_BUILD_VERSION=0",
		"KBUILD_BUILD_TIMESTAMP=now",
		"KBUILD_BUILD_USER=syzkaller",
		"KBUILD_BUILD_HOST=syzkaller",
		"KERNELVERSION=syzkaller",
		"LOCALVERSION=-syzkaller",
	)
	_, err := osutil.Run(time.Hour, cmd)
	return err
}

func (a android) runMake(dir string, params Params, extraArgs ...string) error {
	// LIZ TODO: Not sure why target isn't working
	// extraArgs = append([]string{"CROSS_COMPILE=x86_64-linux-gnu-", "CC=clang", "LD=ld.lld"}, extraArgs...)
	return a.runMakeImpl(dir, params.TargetArch, params.Compiler, params.Linker, params.Ccache, params.KernelDir, extraArgs)
}

func (a android) writeFile(file string, data []byte) error {
	if err := osutil.WriteFile(file, data); err != nil {
		return err
	}
	return osutil.SandboxChown(file)
}

func (a android) clean(kernelDir, targetArch string) error {
	return runMakeImpl(targetArch, "", "", "", kernelDir, []string{"distclean"})
}
