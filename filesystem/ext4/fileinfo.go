package ext4

import (
	"io/fs"
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
	Stat    *syscall.Stat_t
	sys     *StatT
}

type StatT struct {
	UID   uint32
	GID   uint32
	Major uint32
	Minor uint32
	Ino   uint32
	Nlink uint16
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

func modeToFileMode(mode uint16) fs.FileMode {
	var m fs.FileMode

	// File type bits
	switch mode & 0xF000 {
	case 0x4000:
		m |= fs.ModeDir
	case 0xA000:
		m |= fs.ModeSymlink
	case 0x8000:
		// regular file → nothing special
	case 0x2000:
		m |= fs.ModeCharDevice
	case 0x6000:
		m |= fs.ModeDevice
	case 0x1000:
		m |= fs.ModeNamedPipe
	case 0xC000:
		m |= fs.ModeSocket
	default:
		m |= fs.ModeIrregular
	}

	// Special bits
	if mode&0x0800 != 0 {
		m |= fs.ModeSetgid
	}
	if mode&0x1000 != 0 {
		m |= fs.ModeSetuid
	}
	if mode&0x0400 != 0 {
		m |= fs.ModeSticky
	}

	// Permission bits (same layout as in UNIX)
	m |= fs.FileMode(mode & 0x01FF)

	return m
	//return fi.sys
}
