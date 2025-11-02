//go:build linux && arm64

package ext4

import (
	"os"
	"syscall"
	"time"
)

// FileInfo represents the information for an individual file
// it fulfills os.FileInfo interface
type FileInfo struct {
	modTime time.Time
	mode    os.FileMode
	name    string
	size    int64
	isDir   bool
	Stat    syscall.Stat_t
}

func NewFileInfo(modTime time.Time, in *inode, e *directoryEntry) *FileInfo{
	return &FileInfo{
		modTime: modTime,
		name:    e.filename,
		size:    int64(in.size),
		isDir:   e.fileType == dirFileTypeDirectory,
		Stat: syscall.Stat_t{
			Ino:   uint64(e.inode),
			Nlink: uint32(in.hardLinks),
			//Mode:      uint32
			Uid: in.owner,
			Gid: in.group,
			//X__pad0   int32
			//Rdev      uint64
			Size: in.size,
			//Blksize:   int64
			Blocks: int64(in.blocks),
			Atim:   syscall.Timespec{Nsec: in.accessTime.UnixNano()},
			Mtim:   syscall.Timespec{Nsec: in.modifyTime.UnixNano()},
			Ctim:   syscall.Timespec{Nsec: in.createTime.UnixNano()},
		},

	}
}

// IsDir abbreviation for Mode().IsDir()
func (fi *FileInfo) IsDir() bool {
	return fi.isDir
}

// ModTime modification time
func (fi *FileInfo) ModTime() time.Time {
	return fi.modTime
}

// Mode returns file mode
func (fi *FileInfo) Mode() os.FileMode {
	return fi.mode
}

// Name base name of the file
//
//	will return the long name of the file. If none exists, returns the shortname and extension
func (fi *FileInfo) Name() string {
	return fi.name
}

// Size length in bytes for regular files
func (fi *FileInfo) Size() int64 {
	return fi.size
}

// Sys underlying data source - not supported yet and so will return nil
func (fi *FileInfo) Sys() interface{} {
	return fi.Stat
}
