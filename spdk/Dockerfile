FROM chubaofs/cfs-base:1.1 

RUN \ 
    apt-get -y install numactl libnuma-dev ninja-build \
                libncurses5-dev libncursesw5-dev uuid-dev libssl-dev libaio-dev libcunit1-dev  && \
    wget https://www.python.org/ftp/python/3.6.5/Python-3.6.5.tgz && \
    tar -xzvf Python-3.6.5.tgz && \
    cd Python-3.6.5 && ./configure --enable-optimizations && \
    make -j8 build_all && make install && \
    cd ../ && rm -rf Python-3.6.5 && \
    pip3 install meson && \
    rm -rf /var/lib/apt/lists/* && \
    apt-get clean


