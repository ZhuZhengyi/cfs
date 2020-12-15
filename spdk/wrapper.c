// blobfs.c
//

#include <pthread.h>

#include "spdk/bdev_module.h"
#include "spdk/blob.h"
#include "spdk/blob_bdev.h"
#include "spdk/blobfs.h"
#include "spdk/env.h"
#include "spdk/event.h"
#include "spdk/log.h"
#include "spdk/stdinc.h"

#include "spdk_internal/thread.h"

#include "wrapper.h"

#ifdef __cplusplus
extern "C" {
#endif

const char* SPDK_APP_NAME = "goblobfs";
uint32_t g_lcore = 0;
uint64_t g_cache_size = 128;            //mb
struct spdk_thread* g_thread;
pthread_t g_tid;
volatile bool g_spdk_ready = false;
volatile bool g_spdk_start_failure = false;

TAILQ_HEAD(, blobfs_bdev_info) g_bdev_list
	= TAILQ_HEAD_INITIALIZER(g_bdev_list);

static struct blobfs_bdev_info *
_bdev_list_find(const char *name)
{
    struct blobfs_bdev_info *bdev, *tmp;
    TAILQ_FOREACH_SAFE(bdev, &g_bdev_list, link, tmp) {
        if (strcmp(name, bdev->name) == 0) {
            return bdev;
        }
    }

    return NULL;
}

static void
__fs_unload_cb(__attribute__((unused)) void *ctx,
        __attribute__((unused)) int fserrno)
{
	assert(fserrno == 0);

}

static void
_bdev_event_cb(enum spdk_bdev_event_type type, __attribute__((unused)) struct spdk_bdev *bdev,
		   __attribute__((unused)) void *event_ctx)
{
	printf("Unsupported bdev event: type %d\n", type);
}

static void
__spdk_fs_load_cb(void *ctx,
	    struct spdk_filesystem *fs,
        int fserrno)
{
    blobfs_bdev_info_t *dev_info = (blobfs_bdev_info_t*)ctx;
	if (fserrno) {
		SPDK_ERRLOG("Failed to load blobfs on bdev %s: errno %d\n", dev_info->name, fserrno);
        return;
	}
    dev_info->fs = fs;
}

static void
__call_fn(void *arg1, void *arg2)
{
	fs_request_fn fn;

	fn = (fs_request_fn)arg1;
	fn(arg2);
}

static void
__send_request(fs_request_fn fn, void *arg)
{
	struct spdk_event *event;
	//g_lcore = spdk_env_get_first_core();
	event = spdk_event_allocate(g_lcore, __call_fn, (void *)fn, arg);
	spdk_event_call(event);
}

static void
__spdk_blobfs_bdev_detect_cb_complete(void *cb_arg, int fserrno)
{
    int* fs_err = cb_arg;
    *fs_err = fserrno;
}

static void
__spdk_blobfs_bdev_create_cb_complete(void *cb_arg, int fserrno)
{
    int* fs_err = cb_arg;
    *fs_err = fserrno;
}

/* Dummy bdev module used to to claim bdevs. */
static struct spdk_bdev_module blobfs_bdev_module = {
	.name	= "blobfs",
};


/*
 * open blob dev fs info
 * */
blobfs_bdev_info_t*
open_bdev(const char* bdev_name)
{
	int rc = 0;

    struct blobfs_bdev_info *dev_info ;
    dev_info = _bdev_list_find(bdev_name);
    if (dev_info != NULL ) {
		SPDK_INFOLOG(blobfs_bdev, "Failed to create a blobstore block device from bdev (%s)",
			     bdev_name);
        return NULL;
    }
    dev_info = calloc(1, sizeof(blobfs_bdev_info_t));
	if (dev_info == NULL) {
		SPDK_ERRLOG("Failed to allocate ctx.\n");
		return NULL;
	}

    //detect bs
	rc = spdk_bdev_create_bs_dev_ext(bdev_name, _bdev_event_cb, NULL, &dev_info->bdev);
	if (rc != 0) {
		SPDK_INFOLOG(blobfs_bdev, "Failed to create a blobstore block device from bdev (%s)",
			     bdev_name);

        goto err;
	}

    rc = spdk_bs_bdev_claim(dev_info->bdev, &blobfs_bdev_module);
    if (rc != 0) {
        SPDK_INFOLOG(blobfs_bdev, "store block device from bdev (%s)",
	     bdev_name);
         dev_info->bdev->destroy(dev_info->bdev);
        goto err;
    }

    dev_info->name = bdev_name;

    spdk_fs_load(dev_info->bdev, __send_request, __spdk_fs_load_cb, dev_info);

    struct spdk_fs_thread_ctx * ctx = spdk_fs_alloc_thread_ctx(dev_info->fs);
    if (ctx==NULL) {
        //
		SPDK_ERRLOG("Failed to allocate ctx.\n");
        goto err;
    }

    dev_info->ctx=ctx;

    //add dev into tailq
    TAILQ_INSERT_TAIL(&g_bdev_list, dev_info, link);

    return dev_info;

err:
    if (dev_info != NULL) {
        free(dev_info);
    }
    return NULL;
}

int
_close_bdev(struct blobfs_bdev_info *dev_info)
{
    spdk_fs_unload(dev_info->fs, __fs_unload_cb, NULL);

    //remove dev from tailq
    TAILQ_REMOVE(&g_bdev_list, dev_info, link);
    free(dev_info);

    return 0;
}

int
close_bdev(const char* dev_name)
{
    struct blobfs_bdev_info *dev_info = _bdev_list_find(dev_name);
    if (dev_info != NULL) {
        return _close_bdev(dev_info);
    }
    return 0;
}

static void
__spdk_startup_cb(__attribute__((unused)) void *arg)
{
    g_thread = spdk_thread_create("goblobfs", NULL);
    spdk_set_thread(g_thread);

	g_spdk_ready = true;
}

static void
__spdk_shutdown_cb(void)
{
    struct blobfs_bdev_info *bdev, *bdev_tmp;
    // free resource
    TAILQ_FOREACH_SAFE(bdev, &g_bdev_list, link, bdev_tmp) {
        _close_bdev(bdev);
    }
}

static void *
spdk_run(void *opts)
{
	g_lcore = spdk_env_get_first_core();

	spdk_fs_set_cache_size(g_cache_size);

    int rc = spdk_app_start((struct spdk_app_opts*)opts, __spdk_startup_cb, NULL);
    if (rc) {
        g_spdk_start_failure = true;
    } else {
        spdk_app_fini();
    }

    pthread_exit(NULL);
}

int
init_spdk(const char* config, const char* cpu_mask, int cache_size)
{
    assert(config != NULL && cpu_mask != NULL);

    struct spdk_app_opts opts = {};

	spdk_app_opts_init(&opts);
	opts.name = SPDK_APP_NAME;
	opts.json_config_file = config;
	opts.reactor_mask = cpu_mask;
	opts.shutdown_cb = __spdk_shutdown_cb;
    if (cache_size > 0) {
        g_cache_size = cache_size;
    }

    int rc = pthread_create(&g_tid, NULL, spdk_run, &opts);
    if (rc) {
        goto err;
    }
	while (!g_spdk_ready && !g_spdk_start_failure)
		;

    return rc;
err:

    return rc;
}

void
release_spdk()
{
    spdk_app_stop(0);
    //spdk_app_start_shutdown();
    if (g_spdk_ready) {
      pthread_join(g_tid, NULL);
    }
}



#ifdef __cplusplus
}
#endif
