#/bin/bash

cd /syzkaller/managers/ci2-cuttlefish-5-10/kernel-build-test/common
export PATH=$PATH:/syzkaller/managers/ci2-cuttlefish-5-10/kernel-build-test/prebuilts/kernel-build-tools/linux-x86/bin/
export KBUILD_BUILD_VERSION=0, KBUILD_BUILD_TIMESTAMP=now, KBUILD_BUILD_USER=syzkaller, KBUILD_BUILD_HOST=syzkaller, KERNELVERSION=syzkaller, LOCALVERSION=-syzkaller
export ROOT_DIR=/syzkaller/managers/ci2-cuttlefish-5-10/kernel-build-test
export KERNEL_DIR=common
export OUT_DIR=${ROOT_DIR}/${KERNEL_DIR}
export MODULES_STAGING_DIR=${OUT_DIR}/staging
export CC_LD_ARG="-j8 ARCH=x86_64 CROSS_COMPILE=x86_64-linux-gnu- CC=clang LD=ld.lld"

#Source input config file
source $1

source ${MODULES_ENV}

make ${CC_LD_ARG} O=${OUT_DIR} mrproper

cp ${KERNEL_CONFIG} .config

make ${CC_LD_ARG} O=${OUT_DIR} oldconfig

make ${CC_LD_ARG} O=${OUT_DIR} bzImage modules prepare-objtool

make ${CC_LD_ARG} O=${OUT_DIR} INSTALL_MOD_PATH=${MODULES_STAGING_DIR} modules_install

# Make virtual modules
make ${CC_LD_ARG} O=${OUT_DIR} mrproper

cp ${MODULES_CONFIG} .config

make ${CC_LD_ARG} O=${OUT_DIR} oldconfig

make ${CC_LD_ARG} O=${OUT_DIR} bzImage modules prepare-objtool

make ${CC_LD_ARG} O=${OUT_DIR} INSTALL_MOD_PATH=${MODULES_STAGING_DIR} modules_install

cd ..

make ${CC_LD_ARG} -C ${EXT_MODULES} M=../${EXT_MODULES} KERNEL_SRC=${OUT_DIR} O=${OUT_DIR} INSTALL_MOD_PATH=${MODULES_STAGING_DIR}

make ${CC_LD_ARG} -C ${EXT_MODULES} M=../${EXT_MODULES} KERNEL_SRC=${OUT_DIR} O=${OUT_DIR} INSTALL_MOD_PATH=${MODULES_STAGING_DIR} modules_install

zip_kernel_headers

create_initramfs
