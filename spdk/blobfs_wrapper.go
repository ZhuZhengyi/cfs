//

package spdk

// CGO_CFLAGS & CGO_LDFLAGS env vars can be used
// to specify additional dirs.

/*
#cgo CFLAGS:  -I .  -Ispdk/include -Ispdk/dpdk/include
#cgo LDFLAGS: -Llib -lwrapper -lspdk -lnuma -lrt -ldl -luuid -lpthread

#include "stdlib.h"
#include "spdk/stdinc.h"
#include "spdk/blobfs.h"
#include "spdk/env.h"
#include "wrapper.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

const (
	DefaultSpdkJsonCfg   = "spdk.json"
	DefaultSpdkCpuMask   = "0x01"
	DefaultSpdkCacheSize = 256
)

var (
	_init int32 = 0
)

func InitSpdk(config, cpu_mask string, cache_sz uint64) error {

	if !atomic.CompareAndSwapInt32(&_init, 0, 1) {
		return nil
	}
	if config == "" {
		config = DefaultSpdkJsonCfg
	}
	if cpu_mask == "" {
		cpu_mask = DefaultSpdkCpuMask
	}
	if cache_sz == 0 {
		cache_sz = DefaultSpdkCacheSize
	}

	cConfig := C.CString(config)
	defer C.free(unsafe.Pointer(cConfig))

	cCpuMask := C.CString(cpu_mask)
	defer C.free(unsafe.Pointer(cCpuMask))

	if rc := C.init_spdk(cConfig, cCpuMask, C.int(cache_sz)); rc != 0 {
		return Rc2err("init_spdk error", rc)
	}

	return nil
}

func ReleaseSpdk() {
	C.release_spdk()
}

type Bdev struct {
	name      string
	TotalSize uint64
	FreeSize  uint64
	dev       *C.struct_spdk_bs_dev
	fs        *C.struct_spdk_filesystem
	ctx       *C.struct_spdk_fs_thread_ctx
}

// load blob disk blobfs
func OpenBdev(name string) (*Bdev, error) {
	cName := C.CString(name)

	defer C.free(unsafe.Pointer(cName))

	devInfoPtr := C.open_bdev(cName)
	defer func() {
		if devInfoPtr != nil {
			C.free(unsafe.Pointer(devInfoPtr))
		}
	}()

	if devInfoPtr == nil {
		return nil, Rc2err("OpenBdev ", 1)
	}

	return &Bdev{
		name: name,
		dev:  devInfoPtr.bdev,
		fs:   devInfoPtr.fs,
		ctx:  devInfoPtr.ctx,
	}, nil
}

// unload blobfs
func CloseBdev(bdev *Bdev) error {
	cName := C.CString(bdev.name)
	defer C.free(unsafe.Pointer(cName))

	C.close_bdev(cName)

	return nil
}

type BlobFile struct {
	*Bdev
	name string
	f    *C.struct_spdk_file
}

// find
func (dev *Bdev) GetFilesNameWithPrefix(prefix string) ([]string, error) {
	cPrefix := C.CString(prefix)
	defer C.free(unsafe.Pointer(cPrefix))

	files := make([]string, 0)
	for cIter := C.spdk_fs_iter_first(dev.fs); cIter != nil; cIter = C.spdk_fs_iter_next(cIter) {
		if C.spdk_fs_iter_has_prefix(dev.fs, cIter, cPrefix) != 0 {
			cName := C.spdk_fs_iter_get_name(cIter)
			files = append(files, C.GoString(cName))
		}
	}

	return files, nil
}

func (dev *Bdev) OpenFile(name string, flag int, perm os.FileMode) (*BlobFile, error) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	var cFile *C.struct_spdk_file

	//int spdk_fs_open_file(struct spdk_filesystem *fs, struct spdk_fs_thread_ctx *ctx,
	//        const char *name, uint32_t flags, struct spdk_file **file);
	if n := C.spdk_fs_open_file(dev.fs, dev.ctx, cName, C.uint(flag), &cFile); n != 0 {
		return nil, Rc2err("open file error: ", n)
	}

	var bf = &BlobFile{
		Bdev: dev,
		name: name,
		f:    cFile,
	}

	return bf, nil
}

func (fs *Bdev) RemoveFile(name string) error {
	fname := C.CString(name)
	defer C.free(unsafe.Pointer(fname))

	//int spdk_fs_delete_file(struct spdk_filesystem *fs, struct spdk_fs_thread_ctx *ctx,
	//          const char *name);
	if rc := C.spdk_fs_delete_file(fs.fs, fs.ctx, fname); rc != 0 {
		return Rc2err("RemoveFile error ", rc)
	}

	return nil
}

func (dev *Bdev) FsStat() error {
	total := C.spdk_fs_total_size(dev.fs)
	if total <= 0 {
		return errors.New("")
	}
	atomic.StoreUint64(&dev.TotalSize, uint64(total))
	free := C.spdk_fs_free_size(dev.fs)
	if free <= 0 {
		return errors.New("")
	}
	atomic.StoreUint64(&dev.FreeSize, uint64(free))

	return nil
}

func (bdev *Bdev) FileSize(name string) (int64, error) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	var stat C.struct_spdk_file_stat
	if rc := C.spdk_fs_file_stat(bdev.fs, bdev.ctx, cName, &stat); rc != 0 {
		return 0, Rc2err("", rc)
	}

	return int64(stat.size), nil
}

var (
//_ storage.ExtentFile = (*blobFile)(nil)
)

func (f *BlobFile) Name() string {
	fileName := C.spdk_file_get_name(f.f)
	return C.GoString(fileName)
}

func (f *BlobFile) Close() error {
	//int spdk_file_close(struct spdk_file *file, struct spdk_fs_thread_ctx *ctx);
	if rc := C.spdk_file_close(f.f, f.ctx); rc != 0 {
		return Rc2err("Close file error ", rc)
	}
	return nil
}

type BlobFileInfo struct {
	*BlobFile
	size    int64
	mode    os.FileMode
	modTime time.Time
	sys     syscall.Stat_t
}

func (fi *BlobFileInfo) Size() int64 {
	return fi.size
}

func (fi *BlobFileInfo) Mode() os.FileMode {
	return fi.mode
}

func (fi *BlobFileInfo) ModTime() time.Time {
	return fi.modTime
}

func (fi *BlobFileInfo) Sys() interface{} {
	return nil
}

func (fi *BlobFileInfo) IsDir() bool {
	return false
}

func (f *BlobFile) Stat() (os.FileInfo, error) {

	fi := &BlobFileInfo{BlobFile: f}
	if size, err := f.FileSize(f.name); err != nil {
		return nil, err
	} else {
		fi.size = size
	}

	return fi, nil
}

func (f *BlobFile) Write(b []byte) (n int, err error) {

	payload := C.CBytes(b)
	defer C.free(payload)
	length := len(b)
	//int spdk_file_write(struct spdk_file *file, struct spdk_fs_thread_ctx *ctx,
	//          void *payload, uint64_t offset, uint64_t length);
	if rc := C.spdk_file_write_append(f.f, f.ctx, payload, C.ulong(length)); rc <= 0 {
		return 0, Rc2err("write to error", rc)
	}

	return n, nil
}

func (f *BlobFile) WriteAt(b []byte, off int64) (n int, err error) {
	//int spdk_file_randomwrite(struct spdk_file *file, struct spdk_fs_thread_ctx *ctx,
	//        void *payload, uint64_t offset, uint64_t length);
	payload := C.CBytes(b)
	defer C.free(payload)
	var (
		length = len(b)
	)

	if n := C.spdk_file_randomwrite(f.f, f.ctx, payload, C.ulong(off), C.ulong(length)); n <= 0 {
		return int(n), Rc2err("", n)
	}

	return n, nil
}

func (f *BlobFile) ReadAt(b []byte, off int64) (n int, err error) {
	payload := C.CBytes(b)
	defer C.free(payload)

	var (
		length = len(b)
	)

	//int64_t spdk_file_read(struct spdk_file *file, struct spdk_fs_thread_ctx *ctx,
	//             void *payload, uint64_t offset, uint64_t length);
	if rc := C.spdk_file_read(f.f, f.ctx, payload, C.ulong(off), C.ulong(length)); rc <= 0 {
		return int(rc), Rc2err("read error", 1)
	}

	return 0, nil
}

func (f *BlobFile) Sync() (err error) {
	//int spdk_file_sync(struct spdk_file *file, struct spdk_fs_thread_ctx *ctx);
	if rc := C.spdk_file_sync(f.f, f.ctx); rc != 0 {
		return Rc2err("sync", rc)
	}

	return nil
}

func (f *BlobFile) Seek(offset int64, whence int) (ret int64, err error) {
	//TODO: int spdk_file_seek(struct spdk_file *file, struct spdk_fs_thread_ctx *ctx);
	/*
	   if rc := C.spdk_file_seek(f.f, f.ctx, offset, whence); rc <= 0 {
	       return 0, Rc2err("seek", rc)
	   } else {
	       return rc, nil
	   }
	*/
	return 0, nil
}

func (f *BlobFile) Fallocate(mode uint32, off int64, length int64) (err error) {
	//int spdk_file_fallocate(struct spdk_file *file, struct spdk_fs_thread_ctx *ctx, int mode,
	//      uint64_t offset, uint64_t length);
	if n := C.spdk_file_fallocate(f.f, f.ctx, C.int(mode), C.ulong(off), C.ulong(length)); n != 0 {
		return Rc2err("Fallocate", n)
	}

	return nil
}

func (f *BlobFile) Ftruncate(length int64) (err error) {
	// int spdk_file_truncate(struct spdk_file *file, struct spdk_fs_thread_ctx *ctx,
	//             uint64_t length);
	if n := C.spdk_file_truncate(f.f, f.ctx, C.ulong(length)); n != 0 {
		return Rc2err("Fallocate", n)
	}

	return nil
}

func Rc2err(label string, rc C.int) error {
	if rc != 0 {
		if rc < 0 {
			rc = -rc
		}
		return fmt.Errorf("%s: %d", label, rc) // e
	}
	return nil
}
