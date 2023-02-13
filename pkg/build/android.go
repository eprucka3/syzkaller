// Copyright 2022 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package build

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"time"
	"github.com/google/syzkaller/pkg/log"
	"github.com/google/syzkaller/pkg/osutil"
)

type android struct{}

var ccCompilerRegexp = regexp.MustCompile(`#define\s+CONFIG_CC_VERSION_TEXT\s+"(.*)"`)

func (a android) readCompiler(kernelDir string) (string, error) {
	bytes, err := ioutil.ReadFile(filepath.Join("out", "mixed", "device-kernel", "private", "gs-google", "include", "generated", "autoconf.h"))
	if err != nil {
		return "", err
	}
	result := ccCompilerRegexp.FindSubmatch(bytes)
	if result == nil {
		return "", fmt.Errorf("include/generated/autoconf.h does not contain build information")
	}
	return string(result[1]), nil
}

func (a android) build(params Params) (ImageDetails, error) {
	var details ImageDetails
	if params.CmdlineFile != "" {
		return details, fmt.Errorf("cmdline file is not supported for android images")
	}
	if params.SysctlFile != "" {
		return details, fmt.Errorf("sysctl file is not supported for android images")
	}

	// Build kernel.
	// Compiler should match the build for the device (e.g. slider, bluejay...)
	cmd := osutil.Command(fmt.Sprintf("./build_%v.sh", params.Compiler))
	cmd.Dir = params.KernelDir
	defconfigFragment := filepath.Join("private", "gs-google", fmt.Sprintf("build.config.%v.kasan", params.Compiler))
	buildTarget := fmt.Sprintf("%v_gki_kasan", params.Compiler)
	cmd.Env = append(cmd.Env, "OUT_DIR=out", "DIST_DIR=dist", fmt.Sprintf("GKI_DEFCONFIG_FRAGMENT=%v", defconfigFragment), fmt.Sprintf("BUILD_TARGET=%v", buildTarget))
	log.Logf(0, "LIZ_TESTING: cmd: %v", cmd.Args)
	log.Logf(0, "LIZ_TESTING: dir: %v", cmd.Dir)
	
	if _, err := osutil.Run(time.Hour, cmd); err != nil {
		return details, fmt.Errorf("failed to build kernel: %s", err)
	}
	

	// Zip kernel images.
	buildDistDir := filepath.Join(params.KernelDir, "dist")
	dtboImage := filepath.Join(buildDistDir, "dtbo.img")
	vendorBootImage := filepath.Join(buildDistDir, "vendor_boot.img")
	bootImage := filepath.Join(buildDistDir, "boot.img")
	vendorDlkmImage := filepath.Join(buildDistDir, "vendor_dlkm.img")
	vmlinux := filepath.Join(buildDistDir, "vmlinux")
	config := filepath.Join(params.KernelDir, "out", "mixed", "device-kernel", "private", "gs-google", ".config")

	kernelImagesName := "kernel-images.tar.gz"
	kernelImages := filepath.Join(buildDistDir, kernelImagesName)

	cmd = osutil.Command("tar", "-czf", kernelImages, dtboImage, vendorBootImage, bootImage, vendorDlkmImage)
	cmd.Dir = buildDistDir
	if _, err := osutil.Run(time.Minute, cmd); err != nil {
		return details, fmt.Errorf("failed to zip kernel images: %s", err)
	}

	var err error
	details.CompilerID, err = a.readCompiler(params.KernelDir)

	if err := embedFiles(params, func(mountDir string) error {
		homeDir := filepath.Join(mountDir, "root")

		if err := osutil.CopyFile(kernelImages, filepath.Join(homeDir, kernelImagesName)); err != nil {
			return fmt.Errorf("failed to copy kernel images: %v", err)
		}
		if err := osutil.CopyFile(vmlinux, filepath.Join(homeDir, "vmlinux")); err != nil {
			return fmt.Errorf("failed to copy vmlinux: %v", err)
		}

		return nil
	}); err != nil {
		return details, fmt.Errorf("failed to embed files: %v", err)
	}

	if err := osutil.CopyFile(vmlinux, filepath.Join(params.OutputDir, "obj", "vmlinux")); err != nil {
		return details, fmt.Errorf("failed to copy vmlinux: %v", err)
	}
	if err := osutil.CopyFile(config, filepath.Join(params.OutputDir, "obj", "kernel.config")); err != nil {
		return details, fmt.Errorf("failed to copy kernel config: %v", err)
	}

	details.Signature, err = elfBinarySignature(vmlinux, params.Tracer)
	if err != nil {
		return details, fmt.Errorf("failed to generate signature: %s", err)
	}

	return details, nil
}

func (a android) clean(kernelDir, targetArch string) error {
	if err := osutil.RemoveAll(filepath.Join(kernelDir, "out")); err != nil {
		return fmt.Errorf("failed to clean 'out' directory: %v", err)
	}
	if err := osutil.RemoveAll(filepath.Join(kernelDir, "dist")); err != nil {
		return fmt.Errorf("failed to clean 'dist' directory: %v", err)
	}
	return nil
}
