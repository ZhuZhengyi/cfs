#!/usr/bin/env bash

SrcPath=/go/src/github.com/chubaofs/chubaofs

mkdir -p /go/src/github.com/chubaofs/chubaofs/docker/bin;
failed=0

#apt install -y libncurses5-dev libncursesw5-dev libnuma-dev ninja-build uuid-dev libssl-dev libaio-dev libcunit1-dev
#if [ ! `type pip3 >/dev/null 2>&1` ] ; then
#    wget https://www.python.org/ftp/python/3.6.5/Python-3.6.5.tgz
#    tar -xzvf Python-3.6.5.tgz
#    pushd Python-3.6.5
#    ./configure --enable-optimizations
#    make -j8 build
#    make install
#    popd
#fi
#pip3 install meson

pushd $SrcPath/spdk >/dev/null
mkdir -p $SrcPath/spdk/lib
make all
popd >/dev/null

pushd $SrcPath/cmd >/dev/null
echo -n 'Building ChubaoFS Server ... ';
bash ./build.sh &>> /tmp/cfs_build_output
if [[ $? -eq 0 ]]; then
    echo -e "\033[32mdone\033[0m";
    mv cfs-server /go/src/github.com/chubaofs/chubaofs/docker/bin/cfs-server;
else
    echo -e "\033[31mfail\033[0m";
    failed=1
fi
popd >/dev/null

echo -n 'Building ChubaoFS Client ... ' 
pushd $SrcPath/client >/dev/null
bash ./build.sh &>> /tmp/cfs_build_output
if [[ $? -eq 0 ]]; then
    echo -e "\033[32mdone\033[0m";
    mv cfs-client /go/src/github.com/chubaofs/chubaofs/docker/bin/cfs-client;
else
    echo -e "\033[31mfail\033[0m";
    failed=1
fi
popd >/dev/null

echo -n 'Building ChubaoFS CLI    ... '
pushd $SrcPath/cli >/dev/null
bash ./build.sh &>> /tmp/cfs_build_output;
if [[ $? -eq 0 ]]; then
    echo -e "\033[32mdone\033[0m";
    mv cfs-cli /go/src/github.com/chubaofs/chubaofs/docker/bin/cfs-cli;
else
    echo -e "\033[31mfail\033[0m";
    failed=1
fi
popd >/dev/null

if [[ ${failed} -eq 1 ]]; then
    echo -e "\nbuild output:"
    cat /tmp/cfs_build_output;
    exit 1
fi

exit 0
