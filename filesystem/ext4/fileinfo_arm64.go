//go:build linux && arm64

package ext4

import (
	"syscall"
	"time"
)

func NewFileInfo(modTime time.Time, in *inode, e *directoryEntry) *FileInfo {
	return &FileInfo{
		modTime: modTime,
		name:    e.filename,
		size:    int64(in.size),
		isDir:   e.fileType == dirFileTypeDirectory,
		mode:    modeToFileMode(in.mode),
		Stat: &syscall.Stat_t{
			Ino:   uint64(e.inode),
			Nlink: uint32(in.hardLinks),
			Mode:  uint32(in.mode),
			Uid:   in.owner,
			Gid:   in.group,
			//X__pad0   int32
			//Rdev      uint64
			Size: int64(in.size),
			//Blksize:   int64
			Blocks: int64(in.blocks),
			Atim:   syscall.Timespec{Nsec: in.accessTime.UnixNano()},
			Mtim:   syscall.Timespec{Nsec: in.modifyTime.UnixNano()},
			Ctim:   syscall.Timespec{Nsec: in.createTime.UnixNano()},
		},
	}
}
