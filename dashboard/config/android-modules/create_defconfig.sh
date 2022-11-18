#!/bin/bash
KERNEL_DIR=$1
DEFCONFIG=vd_x86_64_gki_defconfig
PRE_DEFCONFIG_CMDS="KCONFIG_CONFIG=${KERNEL_DIR}/arch/x86/configs/${DEFCONFIG} ${KERNEL_DIR}/scripts/kconfig/merge_config.sh -m -r ${ROOT_DIR}/${KERNEL_DIR}/arch/x86/configs/gki_defconfig dashboard/config/android-modules/virtual_device.fragment"

eval $PRE_DEFCONFIG_CMDS

