//go:build linux && arm

package ext4

import (
	"syscall"
)

func newStatT(in *inode) *syscall.Stat_t {
	return &syscall.Stat_t{
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
		Atim:   syscall.Timespec{Nsec: int32(in.accessTime.UnixNano())},
		Mtim:   syscall.Timespec{Nsec: int32(in.modifyTime.UnixNano())},
		Ctim:   syscall.Timespec{Nsec: int32(in.createTime.UnixNano())},
	}
}
