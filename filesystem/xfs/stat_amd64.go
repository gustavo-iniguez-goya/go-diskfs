//go:build linux && amd64

package xfs

import (
	"syscall"
)

// makeStatT creates a syscall.Stat_t from inode information
func makeStatT(inode *Inode, inum uint64) *syscall.Stat_t {
	stat := &syscall.Stat_t{
		Ino:     inum,
		Mode:    uint32(inode.Mode),
		Nlink:   uint64(inode.NLink),
		Uid:     inode.UID,
		Gid:     inode.GID,
		Size:    inode.Size,
		Blksize: 512, // XFS typically uses 512-byte blocks for stat
		Blocks:  inode.NBlocks,
		Atim:    syscall.Timespec{Sec: int64(inode.Atime.Sec), Nsec: int64(inode.Atime.Nsec)},
		Mtim:    syscall.Timespec{Sec: int64(inode.Mtime.Sec), Nsec: int64(inode.Mtime.Nsec)},
		Ctim:    syscall.Timespec{Sec: int64(inode.Ctime.Sec), Nsec: int64(inode.Ctime.Nsec)},
	}
	return stat
}

