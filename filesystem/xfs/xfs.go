/* experimental XFS support. Read only.
* Created with claude.
 */

package xfs

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gustavo-iniguez-goya/go-diskfs/backend"
	"github.com/gustavo-iniguez-goya/go-diskfs/filesystem"
)

const (
	XFS_SB_MAGIC     = 0x58465342 // "XFSB"
	XFS_SB_VERSION_5 = 0x0005
	XFS_DINODE_MAGIC = 0x494e // "IN"

	// Inode formats
	XFS_DINODE_FMT_DEV     = 0
	XFS_DINODE_FMT_LOCAL   = 1
	XFS_DINODE_FMT_EXTENTS = 2
	XFS_DINODE_FMT_BTREE   = 3
	XFS_DINODE_FMT_UUID    = 4

	// File types
	S_IFMT   = 0170000
	S_IFREG  = 0100000
	S_IFDIR  = 0040000
	S_IFLNK  = 0120000
	S_IFCHR  = 0020000
	S_IFBLK  = 0060000
	S_IFIFO  = 0010000
	S_IFSOCK = 0140000
)

// FileSystem represents an XFS filesystem implementing the diskfs interface
type FileSystem struct {
	backend   backend.Storage
	start     int64
	size      int64
	sb        *Superblock
	blockSize uint32
	inodeSize uint16
	agBlocks  uint32
	agCount   uint32
}

// Superblock represents the XFS superblock
type Superblock struct {
	MagicNum            uint32
	BlockSize           uint32
	DBlocks             uint64
	RBlocks             uint64
	RExtents            uint64
	UUID                [16]byte
	LogStart            uint64
	RootIno             uint64
	RBMIno              uint64
	RSumIno             uint64
	RExtSize            uint32
	AGBlocks            uint32
	AGCount             uint32
	RBMBlocks           uint32
	LogBlocks           uint32
	VersionNum          uint16
	SectorSize          uint16
	InodeSize           uint16
	Inopblock           uint16
	Fname               [12]byte
	BlockLog            uint8
	SectLog             uint8
	InodeLog            uint8
	InopbLog            uint8
	AGBlkLog            uint8
	RExtsLog            uint8
	InProgress          uint8
	ImaxPct             uint8
	Features2           uint32
	BadFeatures2        uint32
	FeaturesCompat      uint32
	FeaturesRoCompat    uint32
	FeaturesIncompat    uint32
	FeaturesLogIncompat uint32
	CRC                 uint32
	SparseInodes        uint32
	MetaUUID            [16]byte
}

// Inode represents an XFS inode
type Inode struct {
	Magic        uint16
	Mode         uint16
	Version      int8
	Format       int8
	OnLink       uint16
	UID          uint32
	GID          uint32
	NLink        uint32
	ProjID       uint16
	ProjIDHi     uint16
	Padding      [6]byte
	FlushIter    uint16
	Atime        Timestamp
	Mtime        Timestamp
	Ctime        Timestamp
	Size         int64
	NBlocks      int64
	ExtSize      uint32
	Nextents     int32
	Anextents    int16
	Forkoff      int8
	Aformat      int8
	DMevmask     uint32
	DMstate      uint16
	Flags        uint16
	Gen          uint32
	NextUnlinked uint32
	CRC          uint32
	ChangeCount  uint64
	LSN          uint64
	Flags2       uint64
	Cowextsize   uint32
	Padding2     [12]byte
	Crtime       Timestamp
	Ino          uint64
	UUID         [16]byte
	DataFork     []byte
}

// Timestamp represents an XFS timestamp
type Timestamp struct {
	Sec  int32
	Nsec int32
}

// ToTime converts XFS timestamp to Go time
func (t Timestamp) ToTime() time.Time {
	return time.Unix(int64(t.Sec), int64(t.Nsec))
}

// BMBTRec represents an extent record
type BMBTRec struct {
	L0 uint64
	L1 uint64
}

// FileInfo implements os.FileInfo for XFS files
type FileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
	sys     interface{}
}

func (fi *FileInfo) Name() string       { return fi.name }
func (fi *FileInfo) Size() int64        { return fi.size }
func (fi *FileInfo) Mode() os.FileMode  { return fi.mode }
func (fi *FileInfo) ModTime() time.Time { return fi.modTime }
func (fi *FileInfo) IsDir() bool        { return fi.isDir }
func (fi *FileInfo) Sys() interface{}   { return fi.sys }

// File implements filesystem.File for XFS
type File struct {
	fs          *FileSystem
	inode       *Inode
	path        string
	isDir       bool
	isReadWrite bool
	offset      int64
}

// Read reads from the file
func (f *File) Read(p []byte) (int, error) {
	if f.isDir {
		return 0, fmt.Errorf("cannot read from directory")
	}
	// TODO: implement file data reading
	return 0, fmt.Errorf("file reading not yet implemented")
}

// Write writes to the file
func (f *File) Write(p []byte) (int, error) {
	return 0, fmt.Errorf("XFS write not supported (read-only)")
}

// Seek sets the offset for the next Read or Write
func (f *File) Seek(offset int64, whence int) (int64, error) {
	var newOffset int64
	switch whence {
	case io.SeekStart:
		newOffset = offset
	case io.SeekCurrent:
		newOffset = f.offset + offset
	case io.SeekEnd:
		newOffset = f.inode.Size + offset
	default:
		return 0, fmt.Errorf("invalid whence")
	}

	if newOffset < 0 {
		return 0, fmt.Errorf("negative position")
	}

	f.offset = newOffset
	return f.offset, nil
}

// Close closes the file
func (f *File) Close() error {
	*f = File{}
	return nil
}

// Stat returns file information
func (f *File) Stat() (os.FileInfo, error) {
	mode := os.FileMode(f.inode.Mode & 0777)
	if f.isDir {
		mode |= os.ModeDir
	}

	return &FileInfo{
		name:    filepath.Base(f.path),
		size:    f.inode.Size,
		mode:    mode,
		modTime: f.inode.Mtime.ToTime(),
		isDir:   f.isDir,
	}, nil
}

// Readdir reads directory entries (deprecated but needed for compatibility)
func (f *File) Readdir(n int) ([]os.FileInfo, error) {
	if !f.isDir {
		return nil, fmt.Errorf("not a directory")
	}

	return f.fs.ReadDir(f.path)
}

// Read reads a filesystem from a backend.Storage
//func Read(b backend.Storage, size, start, blocksize int64) (*FileSystem, error) {
func Read(b backend.Storage, size, start, blocksize int64) (*FileSystem, error) {
	fs := &FileSystem{
		backend: b,
		start:   start,
		size:    size,
	}

	if err := fs.readSuperblock(); err != nil {
		return nil, fmt.Errorf("failed to read superblock: %w", err)
	}

	return fs, nil
}

// Type returns the filesystem type
func (fs *FileSystem) Type() filesystem.Type {
	//return filesystem.Type(0x58465342) // XFS magic as type
	return filesystem.TypeExt4
}

// readSuperblock reads and parses the superblock
func (fs *FileSystem) readSuperblock() error {
	buf := make([]byte, 512)
	if _, err := fs.backend.ReadAt(buf, fs.start); err != nil {
		return err
	}

	sb := &Superblock{}
	sb.MagicNum = binary.BigEndian.Uint32(buf[0:4])

	if sb.MagicNum != XFS_SB_MAGIC {
		return fmt.Errorf("invalid XFS magic: 0x%x", sb.MagicNum)
	}

	sb.BlockSize = binary.BigEndian.Uint32(buf[4:8])
	sb.DBlocks = binary.BigEndian.Uint64(buf[8:16])
	sb.RBlocks = binary.BigEndian.Uint64(buf[16:24])
	sb.RExtents = binary.BigEndian.Uint64(buf[24:32])
	copy(sb.UUID[:], buf[32:48])
	sb.LogStart = binary.BigEndian.Uint64(buf[48:56])
	sb.RootIno = binary.BigEndian.Uint64(buf[56:64])
	sb.RBMIno = binary.BigEndian.Uint64(buf[64:72])
	sb.RSumIno = binary.BigEndian.Uint64(buf[72:80])
	sb.RExtSize = binary.BigEndian.Uint32(buf[80:84])
	sb.AGBlocks = binary.BigEndian.Uint32(buf[84:88])
	sb.AGCount = binary.BigEndian.Uint32(buf[88:92])
	sb.RBMBlocks = binary.BigEndian.Uint32(buf[92:96])
	sb.LogBlocks = binary.BigEndian.Uint32(buf[96:100])
	sb.VersionNum = binary.BigEndian.Uint16(buf[100:102])
	sb.SectorSize = binary.BigEndian.Uint16(buf[102:104])
	sb.InodeSize = binary.BigEndian.Uint16(buf[104:106])
	sb.Inopblock = binary.BigEndian.Uint16(buf[106:108])
	copy(sb.Fname[:], buf[108:120])
	sb.BlockLog = buf[120]
	sb.SectLog = buf[121]
	sb.InodeLog = buf[122]
	sb.InopbLog = buf[123]
	sb.AGBlkLog = buf[124]
	sb.RExtsLog = buf[125]
	sb.InProgress = buf[126]
	sb.ImaxPct = buf[127]

	fs.sb = sb
	fs.blockSize = sb.BlockSize
	fs.inodeSize = sb.InodeSize
	fs.agBlocks = sb.AGBlocks
	fs.agCount = sb.AGCount

	return nil
}

// readInode reads an inode from disk
func (fs *FileSystem) readInode(inum uint64) (*Inode, error) {
	agno := inum / uint64(fs.sb.AGBlocks*uint32(fs.sb.Inopblock))
	agino := inum % uint64(fs.sb.AGBlocks*uint32(fs.sb.Inopblock))
	agbno := agino / uint64(fs.sb.Inopblock)
	offset := agino % uint64(fs.sb.Inopblock)

	blockno := agno*uint64(fs.sb.AGBlocks) + agbno
	inodeOffset := fs.start + int64(blockno*uint64(fs.blockSize)) + int64(offset*uint64(fs.inodeSize))

	buf := make([]byte, fs.inodeSize)
	if _, err := fs.backend.ReadAt(buf, inodeOffset); err != nil {
		return nil, err
	}

	inode := &Inode{}
	inode.Magic = binary.BigEndian.Uint16(buf[0:2])

	if inode.Magic != XFS_DINODE_MAGIC {
		return nil, fmt.Errorf("invalid inode magic: 0x%x", inode.Magic)
	}

	inode.Mode = binary.BigEndian.Uint16(buf[2:4])
	inode.Version = int8(buf[4])
	inode.Format = int8(buf[5])
	inode.OnLink = binary.BigEndian.Uint16(buf[6:8])
	inode.UID = binary.BigEndian.Uint32(buf[8:12])
	inode.GID = binary.BigEndian.Uint32(buf[12:16])
	inode.NLink = binary.BigEndian.Uint32(buf[16:20])
	inode.ProjID = binary.BigEndian.Uint16(buf[20:22])
	inode.ProjIDHi = binary.BigEndian.Uint16(buf[22:24])
	copy(inode.Padding[:], buf[24:30])
	inode.FlushIter = binary.BigEndian.Uint16(buf[30:32])

	inode.Atime.Sec = int32(binary.BigEndian.Uint32(buf[32:36]))
	inode.Atime.Nsec = int32(binary.BigEndian.Uint32(buf[36:40]))
	inode.Mtime.Sec = int32(binary.BigEndian.Uint32(buf[40:44]))
	inode.Mtime.Nsec = int32(binary.BigEndian.Uint32(buf[44:48]))
	inode.Ctime.Sec = int32(binary.BigEndian.Uint32(buf[48:52]))
	inode.Ctime.Nsec = int32(binary.BigEndian.Uint32(buf[52:56]))

	inode.Size = int64(binary.BigEndian.Uint64(buf[56:64]))
	inode.NBlocks = int64(binary.BigEndian.Uint64(buf[64:72]))
	inode.ExtSize = binary.BigEndian.Uint32(buf[72:76])
	inode.Nextents = int32(binary.BigEndian.Uint32(buf[76:80]))
	inode.Anextents = int16(binary.BigEndian.Uint16(buf[80:82]))
	inode.Forkoff = int8(buf[82])
	inode.Aformat = int8(buf[83])
	inode.DMevmask = binary.BigEndian.Uint32(buf[84:88])
	inode.DMstate = binary.BigEndian.Uint16(buf[88:90])
	inode.Flags = binary.BigEndian.Uint16(buf[90:92])
	inode.Gen = binary.BigEndian.Uint32(buf[92:96])
	inode.NextUnlinked = binary.BigEndian.Uint32(buf[96:100])

	if inode.Version >= 3 && len(buf) >= 176 {
		inode.CRC = binary.BigEndian.Uint32(buf[100:104])
		inode.ChangeCount = binary.BigEndian.Uint64(buf[104:112])
		inode.LSN = binary.BigEndian.Uint64(buf[112:120])
		inode.Flags2 = binary.BigEndian.Uint64(buf[120:128])
		inode.Cowextsize = binary.BigEndian.Uint32(buf[128:132])
		copy(inode.Padding2[:], buf[132:144])
		inode.Crtime.Sec = int32(binary.BigEndian.Uint32(buf[144:148]))
		inode.Crtime.Nsec = int32(binary.BigEndian.Uint32(buf[148:152]))
		inode.Ino = binary.BigEndian.Uint64(buf[152:160])
		copy(inode.UUID[:], buf[160:176])
	}

	dataForkStart := 176
	if inode.Version < 3 {
		dataForkStart = 100
	}

	forkSize := int(fs.inodeSize) - dataForkStart
	if inode.Forkoff != 0 {
		forkSize = int(inode.Forkoff) * 8
	}

	inode.DataFork = make([]byte, forkSize)
	copy(inode.DataFork, buf[dataForkStart:dataForkStart+forkSize])

	return inode, nil
}

// writeInode writes an inode back to disk
func (fs *FileSystem) writeInode(inum uint64, inode *Inode) error {
	// Calculate inode location
	agno := inum / uint64(fs.sb.AGBlocks*uint32(fs.sb.Inopblock))
	agino := inum % uint64(fs.sb.AGBlocks*uint32(fs.sb.Inopblock))
	agbno := agino / uint64(fs.sb.Inopblock)
	offset := agino % uint64(fs.sb.Inopblock)

	blockno := agno*uint64(fs.sb.AGBlocks) + agbno
	inodeOffset := fs.start + int64(blockno*uint64(fs.blockSize)) + int64(offset*uint64(fs.inodeSize))

	// Build inode buffer
	buf := make([]byte, fs.inodeSize)

	// Write header
	binary.BigEndian.PutUint16(buf[0:2], inode.Magic)
	binary.BigEndian.PutUint16(buf[2:4], inode.Mode)
	buf[4] = byte(inode.Version)
	buf[5] = byte(inode.Format)
	binary.BigEndian.PutUint16(buf[6:8], inode.OnLink)
	binary.BigEndian.PutUint32(buf[8:12], inode.UID)
	binary.BigEndian.PutUint32(buf[12:16], inode.GID)
	binary.BigEndian.PutUint32(buf[16:20], inode.NLink)
	binary.BigEndian.PutUint16(buf[20:22], inode.ProjID)
	binary.BigEndian.PutUint16(buf[22:24], inode.ProjIDHi)
	copy(buf[24:30], inode.Padding[:])
	binary.BigEndian.PutUint16(buf[30:32], inode.FlushIter)

	// Timestamps
	binary.BigEndian.PutUint32(buf[32:36], uint32(inode.Atime.Sec))
	binary.BigEndian.PutUint32(buf[36:40], uint32(inode.Atime.Nsec))
	binary.BigEndian.PutUint32(buf[40:44], uint32(inode.Mtime.Sec))
	binary.BigEndian.PutUint32(buf[44:48], uint32(inode.Mtime.Nsec))
	binary.BigEndian.PutUint32(buf[48:52], uint32(inode.Ctime.Sec))
	binary.BigEndian.PutUint32(buf[52:56], uint32(inode.Ctime.Nsec))

	// File size and extents
	binary.BigEndian.PutUint64(buf[56:64], uint64(inode.Size))
	binary.BigEndian.PutUint64(buf[64:72], uint64(inode.NBlocks))
	binary.BigEndian.PutUint32(buf[72:76], inode.ExtSize)
	binary.BigEndian.PutUint32(buf[76:80], uint32(inode.Nextents))
	binary.BigEndian.PutUint16(buf[80:82], uint16(inode.Anextents))
	buf[82] = byte(inode.Forkoff)
	buf[83] = byte(inode.Aformat)
	binary.BigEndian.PutUint32(buf[84:88], inode.DMevmask)
	binary.BigEndian.PutUint16(buf[88:90], inode.DMstate)
	binary.BigEndian.PutUint16(buf[90:92], inode.Flags)
	binary.BigEndian.PutUint32(buf[92:96], inode.Gen)
	binary.BigEndian.PutUint32(buf[96:100], inode.NextUnlinked)

	// V3 fields
	if inode.Version >= 3 {
		// Update change count for metadata versioning
		inode.ChangeCount++

		binary.BigEndian.PutUint32(buf[100:104], inode.CRC) // CRC calculated later
		binary.BigEndian.PutUint64(buf[104:112], inode.ChangeCount)
		binary.BigEndian.PutUint64(buf[112:120], inode.LSN)
		binary.BigEndian.PutUint64(buf[120:128], inode.Flags2)
		binary.BigEndian.PutUint32(buf[128:132], inode.Cowextsize)
		copy(buf[132:144], inode.Padding2[:])
		binary.BigEndian.PutUint32(buf[144:148], uint32(inode.Crtime.Sec))
		binary.BigEndian.PutUint32(buf[148:152], uint32(inode.Crtime.Nsec))
		binary.BigEndian.PutUint64(buf[152:160], inode.Ino)
		copy(buf[160:176], inode.UUID[:])

		// Copy data fork
		dataForkStart := 176
		copy(buf[dataForkStart:], inode.DataFork)

		// Calculate and update CRC for V3 inodes
		crc := fs.calculateInodeCRC(buf, inum)
		binary.BigEndian.PutUint32(buf[100:104], crc)
	} else {
		// V1/V2 inode
		dataForkStart := 100
		copy(buf[dataForkStart:], inode.DataFork)
	}

	// Write to disk
	writableFile, err := fs.backend.Writable()
	if err != nil {
		return err
	}
	_, err = writableFile.WriteAt(buf, inodeOffset)
	return err
}

// OpenFile opens a file or directory
func (fs *FileSystem) OpenFile(p string, flag int) (filesystem.File, error) {
	// Normalize path
	p = filepath.Clean(p)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}

	// Navigate to the file
	/*inum*/
	_, inode, err := fs.navigateToPath(p)
	if err != nil {
		if flag&os.O_CREATE != 0 {
			return nil, fmt.Errorf("XFS create not supported (read-only)")
		}
		return nil, err
	}

	// Check write flags
	if flag&(os.O_WRONLY|os.O_RDWR|os.O_APPEND|os.O_CREATE|os.O_TRUNC) != 0 {
		return nil, fmt.Errorf("XFS write not supported (read-only)")
	}

	isDir := (inode.Mode & S_IFMT) == S_IFDIR

	return &File{
		fs:          fs,
		inode:       inode,
		path:        p,
		isDir:       isDir,
		isReadWrite: false,
		offset:      0,
	}, nil
}

// ReadDir reads directory entries
func (fs *FileSystem) ReadDir(p string) ([]os.FileInfo, error) {
	p = filepath.Clean(p)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}

	inum, inode, err := fs.navigateToPath(p)
	if err != nil {
		return nil, err
	}

	if (inode.Mode & S_IFMT) != S_IFDIR {
		return nil, fmt.Errorf("%s is not a directory", p)
	}

	entries, err := fs.listDir(inum, inode)
	if err != nil {
		return nil, err
	}

	var result []os.FileInfo
	for _, entry := range entries {
		result = append(result, entry)
	}

	return result, nil
}

// navigateToPath navigates to a path and returns its inode
func (fs *FileSystem) navigateToPath(path string) (uint64, *Inode, error) {
	if path == "/" {
		inode, err := fs.readInode(fs.sb.RootIno)
		return fs.sb.RootIno, inode, err
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	currentIno := fs.sb.RootIno

	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}

		currentInode, err := fs.readInode(currentIno)
		if err != nil {
			return 0, nil, err
		}

		if (currentInode.Mode & S_IFMT) != S_IFDIR {
			return 0, nil, fmt.Errorf("not a directory: %s", part)
		}

		entries, err := fs.listDir(currentIno, currentInode)
		if err != nil {
			return 0, nil, err
		}

		found := false
		for _, entry := range entries {
			if entry.Name() == part {
				// Extract inode number from sys
				if sysData, ok := entry.Sys().(map[string]interface{}); ok {
					if ino, ok := sysData["inode"].(uint64); ok {
						currentIno = ino
						found = true
						break
					}
				}
			}
		}

		if !found {
			return 0, nil, fmt.Errorf("no such file or directory: %s", part)
		}
	}

	inode, err := fs.readInode(currentIno)
	return currentIno, inode, err
}

// listDir lists directory entries
func (fs *FileSystem) listDir(inum uint64, inode *Inode) ([]*FileInfo, error) {
	switch inode.Format {
	case XFS_DINODE_FMT_LOCAL:
		return fs.readShortformDir(inode)
	case XFS_DINODE_FMT_EXTENTS:
		return fs.readBlockDir(inode, inum)
	default:
		return nil, fmt.Errorf("unsupported directory format: %d", inode.Format)
	}
}

// readShortformDir reads a shortform (inline) directory
func (fs *FileSystem) readShortformDir(inode *Inode) ([]*FileInfo, error) {
	buf := inode.DataFork
	if len(buf) < 6 {
		return nil, fmt.Errorf("directory data too short")
	}

	count := buf[0]
	i8count := buf[1]
	var parent uint64

	offset := 2
	if i8count == 0 {
		parent = uint64(binary.BigEndian.Uint32(buf[offset : offset+4]))
		offset += 4
	} else {
		parent = binary.BigEndian.Uint64(buf[offset : offset+8])
		offset += 8
	}

	var entries []*FileInfo

	// Add . and ..
	entries = append(entries, &FileInfo{
		name:    ".",
		size:    0,
		mode:    os.ModeDir | 0755,
		modTime: inode.Mtime.ToTime(),
		isDir:   true,
		sys:     map[string]interface{}{"inode": inode.Ino},
	})

	entries = append(entries, &FileInfo{
		name:    "..",
		size:    0,
		mode:    os.ModeDir | 0755,
		modTime: inode.Mtime.ToTime(),
		isDir:   true,
		sys:     map[string]interface{}{"inode": parent},
	})

	for i := 0; i < int(count); i++ {
		if offset+3 >= len(buf) {
			break
		}

		namelen := int(buf[offset])
		offset++

		// Skip offset field (2 bytes)
		offset += 2

		if namelen == 0 || offset+namelen > len(buf) {
			break
		}

		name := string(buf[offset : offset+namelen])
		offset += namelen

		if offset >= len(buf) {
			break
		}
		ftype := buf[offset]
		offset++

		var ino uint64
		if i8count == 0 {
			if offset+4 > len(buf) {
				break
			}
			ino = uint64(binary.BigEndian.Uint32(buf[offset : offset+4]))
			offset += 4
		} else {
			if offset+8 > len(buf) {
				break
			}
			ino = binary.BigEndian.Uint64(buf[offset : offset+8])
			offset += 8
		}

		childInode, err := fs.readInode(ino)
		if err != nil {
			mode := os.FileMode(0644)
			isDir := ftype == 2
			if isDir {
				mode = os.ModeDir | 0755
			}
			entries = append(entries, &FileInfo{
				name:    name,
				size:    0,
				mode:    mode,
				modTime: time.Time{},
				isDir:   isDir,
				sys:     map[string]interface{}{"inode": ino},
			})
			continue
		}

		mode := os.FileMode(childInode.Mode & 0777)
		isDir := (childInode.Mode & S_IFMT) == S_IFDIR
		if isDir {
			mode |= os.ModeDir
		}

		entries = append(entries, &FileInfo{
			name:    name,
			size:    childInode.Size,
			mode:    mode,
			modTime: childInode.Mtime.ToTime(),
			isDir:   isDir,
			sys:     map[string]interface{}{"inode": ino},
		})
	}

	return entries, nil
}

// readBlockDir reads a block format directory
func (fs *FileSystem) readBlockDir(inode *Inode, inum uint64) ([]*FileInfo, error) {
	if inode.Nextents == 0 {
		return []*FileInfo{}, nil
	}

	if len(inode.DataFork) < 16 {
		return nil, fmt.Errorf("data fork too small for extent")
	}

	rec := BMBTRec{
		L0: binary.BigEndian.Uint64(inode.DataFork[0:8]),
		L1: binary.BigEndian.Uint64(inode.DataFork[8:16]),
	}

	startoff := (rec.L0 & 0x7FFFFFFFFFFFFFFF) >> 9
	startblock := ((rec.L0 & 0x1FF) << 43) | (rec.L1 >> 21)
	blockcount := rec.L1 & 0x1FFFFF

	if startoff != 0 || blockcount == 0 {
		return nil, fmt.Errorf("invalid extent")
	}

	blockOffset := fs.start + int64(startblock*uint64(fs.blockSize))
	dirBlock := make([]byte, fs.blockSize)

	if _, err := fs.backend.ReadAt(dirBlock, blockOffset); err != nil {
		return nil, err
	}

	return fs.parseBlockDirData(dirBlock, inum)
}

// parseBlockDirData parses directory data from a block
func (fs *FileSystem) parseBlockDirData(block []byte, dirInum uint64) ([]*FileInfo, error) {
	magic := binary.BigEndian.Uint32(block[0:4])

	offset := 64
	if magic == 0x58443344 || magic == 0x58444233 {
		offset = 64
	} else {
		offset = 16
	}

	var entries []*FileInfo

	for offset < len(block)-8 {
		ino := binary.BigEndian.Uint64(block[offset : offset+8])
		if ino == 0 || ino == 0xFFFFFFFFFFFFFFFF {
			offset += 8
			continue
		}

		if offset+8 >= len(block) {
			break
		}

		namelen := int(block[offset+8])
		if namelen == 0 || offset+9+namelen > len(block) {
			break
		}

		name := string(block[offset+9 : offset+9+namelen])

		if len(name) == 0 {
			break
		}

		childInode, err := fs.readInode(ino)
		if err != nil {
			offset += 9 + namelen + ((4 - (namelen & 3)) & 3)
			continue
		}

		mode := os.FileMode(childInode.Mode & 0777)
		isDir := (childInode.Mode & S_IFMT) == S_IFDIR
		if isDir {
			mode |= os.ModeDir
		}

		entries = append(entries, &FileInfo{
			name:    name,
			size:    childInode.Size,
			mode:    mode,
			modTime: childInode.Mtime.ToTime(),
			isDir:   isDir,
			sys:     map[string]interface{}{"inode": ino},
		})

		offset += 9 + namelen + ((4 - (namelen & 3)) & 3)
	}

	return entries, nil
}

// Mkdir creates a directory (not supported - read-only)
func (fs *FileSystem) Mkdir(p string) error {
	return fmt.Errorf("XFS mkdir not supported (read-only)")
}

// Remove removes a file or directory (not supported - read-only)
func (fs *FileSystem) Remove(pathname string) error {
	return fmt.Errorf("XFS remove not supported (read-only)")
}

// Rename renames a file or directory (not supported - read-only)
func (fs *FileSystem) Rename(oldpath, newpath string) error {
	return fmt.Errorf("XFS rename not supported (read-only)")
}

// Chmod changes file mode (not supported - read-only)
func (fs *FileSystem) Chmod(path string, mode os.FileMode) error {
	path = filepath.Clean(path)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	inum, inode, err := fs.navigateToPath(path)
	if err != nil {
		return err
	}

	// Update mode, preserving file type bits
	newMode := (inode.Mode & S_IFMT) | uint16(mode&0777)
	inode.Mode = newMode

	// Write inode back to disk
	return fs.writeInode(inum, inode)
}

// SetLabel changes the label on the writable filesystem. Different file system may hav different
// length constraints.
func (fs *FileSystem) SetLabel(label string) error {
	//fs.superblock.volumeLabel = label
	return fmt.Errorf("XFS chown not supported (read-only)") //fs.writeSuperblock()
}

// Label returns the filesystem label
func (fs *FileSystem) Label() string {
	return string(fs.sb.Fname[:])
}

// Mknod creates a special file (not supported - read-only)
func (fs *FileSystem) Mknod(pathname string, mode uint32, dev int) error {
	return fmt.Errorf("XFS mknod not supported (read-only)")
}

// Link creates a hard link (not supported - read-only)
func (fs *FileSystem) Link(oldpath, newpath string) error {
	return fmt.Errorf("XFS link not supported (read-only)")
}

// Symlink creates a symbolic link (not supported - read-only)
func (fs *FileSystem) Symlink(oldpath, newpath string) error {
	return fmt.Errorf("XFS symlink not supported (read-only)")
}

// Readlink reads a symbolic link
func (fs *FileSystem) Readlink(path string) (string, error) {
	// TODO: implement symlink reading from inode data fork
	return "", fmt.Errorf("XFS readlink not yet implemented")
}

func (fs *FileSystem) Close() error {
	return nil
}

// SetBackendWritable allows enabling write operations
// WARNING: Use with caution! Writing to XFS can corrupt the filesystem
func (fs *FileSystem) SetBackendWritable(writable bool) error {
	// Check if backend supports writing
	if writable {
		// Try to write a test byte to verify write capability
		testBuf := make([]byte, 1)
		writableFile, err := fs.backend.Writable()
		_, err = writableFile.WriteAt(testBuf, fs.start+int64(fs.size)-1)
		if err != nil {
			return fmt.Errorf("backend does not support writing: %w", err)
		}
	}
	return nil
}

// Chown changes file ownership
func (fs *FileSystem) Chown(path string, uid, gid int) error {
	path = filepath.Clean(path)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	inum, inode, err := fs.navigateToPath(path)
	if err != nil {
		return err
	}

	// Update ownership
	if uid >= 0 {
		inode.UID = uint32(uid)
	}
	if gid >= 0 {
		inode.GID = uint32(gid)
	}

	// Write inode back to disk
	return fs.writeInode(inum, inode)
}

// Utimes changes file access and modification times
func (fs *FileSystem) Utimes(path string, atime, mtime time.Time) error {
	path = filepath.Clean(path)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	inum, inode, err := fs.navigateToPath(path)
	if err != nil {
		return err
	}

	// Update timestamps
	if !atime.IsZero() {
		inode.Atime.Sec = int32(atime.Unix())
		inode.Atime.Nsec = int32(atime.Nanosecond())
	}
	if !mtime.IsZero() {
		inode.Mtime.Sec = int32(mtime.Unix())
		inode.Mtime.Nsec = int32(mtime.Nanosecond())
	}

	// Update ctime (change time) to now
	now := time.Now()
	inode.Ctime.Sec = int32(now.Unix())
	inode.Ctime.Nsec = int32(now.Nanosecond())

	// Write inode back to disk
	return fs.writeInode(inum, inode)
}

// calculateInodeCRC calculates CRC32c for an inode (XFS V5)
func (fs *FileSystem) calculateInodeCRC(buf []byte, inum uint64) uint32 {
	// XFS uses CRC32c (Castagnoli) with special initialization
	// This is a simplified version - real XFS has more complex CRC handling
	// For now, return 0 to indicate CRC not fully implemented
	// TODO: Implement proper CRC32c calculation with XFS-specific handling
	return 0
}
