# Makefile
#
#

DOCKER_IMAGE_NAME := chubaofs/cfs-base:1.2-spdk

CC := gcc
AR := ar
GCC_FLAGS := -g -Wshadow -Wall -Wno-missing-braces -Wno-unused-function
GCC_CFLAGS := -Ispdk/include -Ispdk/dpdk/include
GCC_LDFLAGS := -Llib -lspdk -lnuma -lrt -ldl -luuid -lpthread

phony := all
all: build

phony += build
build: lib/libwrapper.a lib/libspdk.a bin/mkfs

lib/libspdk.a:
	@cd spdk && ./configure --disable-tests --disable-unit-tests && make
	$(AR) -M < libspdk.mri

lib/libwrapper.a: wrapper.o
	$(AR) -rcs $@ $<

wrapper.o: wrapper.c
	$(CC) ${GCC_CFLAGS} ${GCC_FLAGS} -c $<

bin/mkfs: mkfs.o
	@mkdir -p bin
	$(CC) -o $@ $< $(GCC_LDFLAGS)

mkfs.o: tools/mkfs.c
	$(CC) $(GCC_CFLAGS) -c $<

phony += docker-build
docker-build:
	docker build -t $(DOCKER_IMAGE_NAME) -f Dockerfile lib

phony += docker-push
docker-push:
	docker image push $(DOCKER_IMAGE_NAME)

phony += clean
clean:
	@$(RM) -rf *.o

phony += dist-clean
dist-clean:
	@$(RM) -rf *.o bin/* lib/*.a

.PHONY: $(phony)

