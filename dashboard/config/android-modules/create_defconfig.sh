#!/bin/bash
ROOT_DIR=$1
KERNEL_DIR=common
DEFCONFIG=vd_x86_64_gki_defconfig
PRE_DEFCONFIG_CMDS="KCONFIG_CONFIG=${ROOT_DIR}/${KERNEL_DIR}/arch/x86/configs/${DEFCONFIG} ${ROOT_DIR}/${KERNEL_DIR}/scripts/kconfig/merge_config.sh -m -r ${ROOT_DIR}/${KERNEL_DIR}/arch/x86/configs/gki_defconfig ${ROOT_DIR}/common-modules/virtual-device/virtual_device.fragment"

eval $PRE_DEFCONFIG_CMDS

