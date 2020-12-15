// goblobfs.h
#ifndef __WRAPPER_H
#define __WRAPPER_H

#include <stdlib.h>

#include "spdk/blob.h"
#include "spdk/blob_bdev.h"
#include "spdk/blobfs.h"
#include "spdk_internal/thread.h"

typedef struct blobfs_bdev_info {
  const char *name;
  struct spdk_bs_dev *bdev;
  struct spdk_filesystem *fs;
  struct spdk_fs_thread_ctx *ctx;
  TAILQ_ENTRY(blobfs_bdev_info) link;
} blobfs_bdev_info_t;

int init_spdk(const char *config, const char *cpu_mask, int cache_size);

void release_spdk();

// static int open_bdev(const char *bdev_name, struct blobfs_bdev_info
// *dev_info);
blobfs_bdev_info_t *open_bdev(const char *bdev_name);

// static int close_bdev(struct blobfs_bdev_info *dev_info);
int close_bdev(const char *dev_name);

int _close_bdev(struct blobfs_bdev_info *dev_info);

#endif
