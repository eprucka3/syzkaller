#!/bin/bash
ROOT_DIR=$1
TARGET_CONFIG=$2

# Syz-kconv turns module configs ("=m") to yes ("=y"), but the module configs in 
# virtual_device.fragment are necessary when installing modules.
# This script finds all configs set to "=m" in virtual_device.fragment and uses
# ${ROOT_DIR}/{KERNEL_DIR}'scripts/config to update configs generated by
# syz-kconf.
MODULE_CONFIGS=$(eval 'grep "=m$" ${ROOT_DIR}/common-modules/virtual-device/virtual_device.fragment | sed -e "s/^/-m /" -e "s/=m//"')

# Update target config with virtual_device.fragment
${ROOT_DIR}/${KERNEL_DIR}/scripts/config --file ${TARGET_CONFIG} ${MODULE_CONFIGS}
