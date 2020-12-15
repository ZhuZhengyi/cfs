#!/bin/bash

BASE_DIR=$(cd $(dirname $BASH_SOURCE); pwd)

export SPDK_REPO=${SPDK_REPO:-"$BASE_DIR/spdk"}
export DPDK_REPO=${DPDK_REPO:-"${SPDK_REPO}/dpdk"}

set -e

mkdir -p $BASE_DIR/lib

if [ ! -e $BASE_DIR/lib/libspdk.a ] ; then
    pushd $SPDK_REPO >/dev/null
    ./configure --disable-tests --disable-unit-tests && make
    popd >/dev/null
    ar -M < libspdk.mri
fi

if [ ! -e $BASE_DIR/lib/libwrapper.a ] ; then
    pushd $BASE_DIR >/dev/null
    GCC_CFLAGS="-I${SPDK_REPO}/include -I${DPDK_REPO}/include"
    GCC_FLAGS="-g -Wshadow -Wall -Wno-missing-braces -Wno-unused-function "
    gcc ${GCC_CFLAGS} ${GCC_FLAGS} -c $BASE_DIR/wrapper.c
    ar -rcs $BASE_DIR/lib/libwrapper.a *.o
    popd >/dev/null
fi
