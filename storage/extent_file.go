// Copyright 2018 The Chubao Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.
//
// extent_file.go
//

package storage

import (
	"os"
	"syscall"

	blobfs "github.com/chubaofs/chubaofs/spdk"
)

//
type ExtentFile interface {
	Name() string
	Write(b []byte) (n int, err error)
	WriteAt(b []byte, off int64) (n int, err error)
	ReadAt(b []byte, off int64) (n int, err error)
	Sync() error
	Seek(offset int64, whence int) (ret int64, err error)
	Stat() (os.FileInfo, error)
	Close() error
	Fallocate(mode uint32, off int64, len int64) (err error)
	Ftruncate(length int64) (err error)
}

var (
//_ ExtentFile = (*blobfs.spdk.BlobFile)(nil)
)

type extentFile struct {
	*os.File
}

var (
	_ ExtentFile = (*extentFile)(nil)
)

func (f *extentFile) Fallocate(mode uint32, off int64, len int64) error {
	return fallocate(int(f.Fd()), mode, off, len)
}

func (f *extentFile) Ftruncate(length int64) (err error) {
	return syscall.Ftruncate(int(f.Fd()), length)
}

type ExtentDev interface {
	OpenDev() *ExtentDev
}

type ExtentFs struct {
	DiskPath string
	FsType   ExtentFsType
	NvmeDev  *blobfs.Bdev
}

func NewExtentFs(diskPath string, fsType ExtentFsType) *ExtentFs {
	extentFS := &ExtentFs{
		DiskPath: diskPath,
		FsType:   fsType,
	}
	if fsType == ExtentFsBlobFS {
		bdev, err := blobfs.OpenBdev(diskPath)
		if err != nil {

		}
		extentFS.NvmeDev = bdev

	}
	return extentFS
}

type ExtentFsType uint8

const (
	ExtentFsStandard ExtentFsType = iota //standard fs file
	ExtentFsBlobFS                       //nvme blobfs file

	DiskDefaultFs = ExtentFsStandard
)

func (fs *ExtentFs) IsBlobFs() bool {
	return fs.FsType == ExtentFsBlobFS
}

// Open os file with path
func (fs *ExtentFs) OpenFile(path string, flag int, perm os.FileMode) (ExtentFile, error) {
	if fs.FsType == ExtentFsStandard {
		if f, err := os.OpenFile(path, flag, perm); err != nil {
			return nil, err
		} else {
			return &extentFile{f}, nil
		}
	} else {
		if f, er := fs.NvmeDev.OpenFile(path, flag, perm); er != nil {
			return nil, er
		} else {
			return f, nil
		}

	}
}

func (fs *ExtentFs) RemoveFile(path string) (err error) {
	if fs.FsType == ExtentFsStandard {
		return os.Remove(path)
	} else {
		return fs.NvmeDev.RemoveFile(path)
	}
}

// Parse extentFsType from string
func ParseExtentFsType(stype string) ExtentFsType {
	switch {
	case stype == "1" || stype == "nvme":
		return ExtentFsBlobFS
	default:
		return ExtentFsStandard
	}
}
