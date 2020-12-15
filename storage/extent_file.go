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

type ExtentFs struct {
	DiskPath string
	FsType   ExtentFsType
}

func NewExtentFs(diskPath string, fsType ExtentFsType) *ExtentFs {
	return &ExtentFs{
		DiskPath: diskPath,
		FsType:   fsType,
	}
}

type ExtentFsType uint8

const (
	ExtentFsStandard ExtentFsType = iota //standard fs file
	ExtentFsBlobFS                       //nvme blobfs file

	DiskDefaultFs = ExtentFsStandard
)

// Open os file with path
func (fs *ExtentFs) OpenFile(path string, flag int, perm os.FileMode) (*extentFile, error) {
	if fs.FsType == ExtentFsStandard {
		if f, err := os.OpenFile(path, flag, perm); err != nil {
			return nil, err
		} else {
			return &extentFile{f}, nil
		}
	} else {
		//TODO: open blobfs file
		//blobfs.OpenFile()
	}

	return nil, nil
}

func (fs *ExtentFs) RemoveFile(path string) (err error) {
	if fs.FsType == ExtentFsStandard {
		return os.Remove(path)
	} else {
		//TODO: remove blobfs file
	}

	return nil
}

// Parse extentFsType from string
func ParseExtentFsType(stype string) ExtentFsType {
	switch {
	case stype == "1" || stype == "blobfs":
		return ExtentFsBlobFS
	default:
		return ExtentFsStandard
	}
}
