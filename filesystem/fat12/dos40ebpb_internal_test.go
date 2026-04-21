package fat12

// dos40ebpb_internal_test.go exercises Dos40EBPB parsing and serialisation.

import (
	"testing"
)

func makeValidDos20Bytes() []byte {
	// 13 bytes: bytesPerSector=512, sectorsPerCluster=1, reservedSectors=1,
	// fatCount=2, rootDirEntries=224, totalSectors=2880, mediaType=0xF0,
	// sectorsPerFat=9
	b := make([]byte, 13)
	b[0], b[1] = 0x00, 0x02   // bytesPerSector = 512 (little-endian)
	b[2] = 0x01               // 1 sector per cluster
	b[3], b[4] = 0x01, 0x00   // 1 reserved sector (little-endian)
	b[5] = 0x02               // fatCount = 2
	b[6], b[7] = 0xE0, 0x00   // 224 root directory entries (little-endian)
	b[8], b[9] = 0x40, 0x0B   // 2880 total sectors (little-endian)
	b[10] = 0xF0              // mediaType (3.5" floppy)
	b[11], b[12] = 0x09, 0x00 // 9 sectors per FAT (little-endian)
	return b
}

func makeValidDos331Bytes() []byte {
	b := make([]byte, 25)
	copy(b[0:13], makeValidDos20Bytes())
	b[13], b[14] = 0x12, 0x00 // 18 sectors per track (little-endian)
	b[15], b[16] = 0x02, 0x00 // heads = 2
	// hiddenSectors = 0 (bytes 17-20)
	// totalSectors32 = 0 (bytes 21-24)
	return b
}

// makeValidDos40EBPBBytes builds a 51-byte long-form Dos40EBPB matching a
// standard 1.44 MB FAT12 floppy layout.
func makeValidDos40EBPBBytes() []byte {
	b := make([]byte, 51)
	copy(b[0:25], makeValidDos331Bytes())
	b[25] = 0x00     // driveNumber (removable)
	b[26] = 0x00     // reservedFlags
	b[27] = longEBPB // extended boot signature (long form)
	b[28] = 0xAD     // volumeSerialNumber (4 bytes, little-endian)
	b[29] = 0xDE
	b[30] = 0xAD
	b[31] = 0xBE
	copy(b[32:43], "NO NAME    ") // volumeLabel (11 bytes)
	copy(b[43:51], "FAT12   ")    // fileSystemType (8 bytes)
	return b
}

func TestDos40EBPBFromBytes(t *testing.T) {
	t.Run("short form (0x28)", func(t *testing.T) {
		b := make([]byte, 32)
		copy(b[0:25], makeValidDos331Bytes())
		b[27] = shortEBPB
		bpb, n, err := Dos40EBPBFromBytes(b)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n != 32 {
			t.Errorf("consumed %d bytes, want 32", n)
		}
		if bpb.ExtBootSignature != shortEBPB {
			t.Errorf("ExtBootSignature = 0x%02X, want 0x%02X", bpb.ExtBootSignature, shortEBPB)
		}
		if bpb.VolumeLabel != "" {
			t.Errorf("short form should have empty VolumeLabel, got %q", bpb.VolumeLabel)
		}
	})

	t.Run("long form (0x29)", func(t *testing.T) {
		b := makeValidDos40EBPBBytes()
		bpb, n, err := Dos40EBPBFromBytes(b)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n != 51 {
			t.Errorf("consumed %d bytes, want 51", n)
		}
		if bpb.ExtBootSignature != longEBPB {
			t.Errorf("ExtBootSignature = 0x%02X, want 0x%02X", bpb.ExtBootSignature, longEBPB)
		}
		if bpb.VolumeLabel != "NO NAME" {
			t.Errorf("VolumeLabel = %q, want %q", bpb.VolumeLabel, "NO NAME")
		}
		if bpb.FileSystemType != "FAT12" {
			t.Errorf("FileSystemType = %q, want %q", bpb.FileSystemType, "FAT12")
		}
	})

	t.Run("too short", func(t *testing.T) {
		_, _, err := Dos40EBPBFromBytes(make([]byte, 10))
		if err == nil {
			t.Error("expected error for too-short input, got nil")
		}
	})

	t.Run("invalid signature", func(t *testing.T) {
		b := makeValidDos40EBPBBytes()
		b[27] = 0x99 // bogus signature
		_, _, err := Dos40EBPBFromBytes(b)
		if err == nil {
			t.Error("expected error for invalid signature, got nil")
		}
	})

	t.Run("long form too short", func(t *testing.T) {
		b := make([]byte, 40) // 0x29 requires 51 bytes
		copy(b[0:25], makeValidDos331Bytes())
		b[27] = longEBPB
		_, _, err := Dos40EBPBFromBytes(b)
		if err == nil {
			t.Error("expected error for long form with only 40 bytes, got nil")
		}
	})
}

func TestDos40EBPBRoundTrip(t *testing.T) {
	t.Run("long form", func(t *testing.T) {
		original := makeValidDos40EBPBBytes()
		bpb, _, err := Dos40EBPBFromBytes(original)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		serialised, err := bpb.ToBytes()
		if err != nil {
			t.Fatalf("ToBytes error: %v", err)
		}
		if len(serialised) != len(original) {
			t.Fatalf("serialised length %d != original %d", len(serialised), len(original))
		}
		for i := range original {
			if serialised[i] != original[i] {
				t.Errorf("byte[%d]: got 0x%02X, want 0x%02X", i, serialised[i], original[i])
			}
		}
	})

	t.Run("short form", func(t *testing.T) {
		b := make([]byte, 32)
		copy(b[0:25], makeValidDos331Bytes())
		b[27] = shortEBPB
		bpb, _, err := Dos40EBPBFromBytes(b)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		serialised, err := bpb.ToBytes()
		if err != nil {
			t.Fatalf("ToBytes error: %v", err)
		}
		if len(serialised) != 32 {
			t.Fatalf("short form should be 32 bytes, got %d", len(serialised))
		}
	})
}

func TestDos40EBPBTotalSectors(t *testing.T) {
	b := makeValidDos40EBPBBytes()
	bpb, _, err := Dos40EBPBFromBytes(b)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// From makeValidDos20Bytes: TotalSectors = 2880, TotalSectors32 = 0.
	// TotalSectors() should prefer TotalSectors32 when non-zero; fall back to 16-bit.
	if got := bpb.TotalSectors(); got != 2880 {
		t.Errorf("TotalSectors() = %d, want 2880", got)
	}

	// Now simulate a large volume where the 16-bit field overflows.
	bpb.Dos331BPB.Dos20BPB.TotalSectors = 0
	bpb.Dos331BPB.TotalSectors32 = 4096000
	if got := bpb.TotalSectors(); got != 4096000 {
		t.Errorf("TotalSectors() with 32-bit field = %d, want 4096000", got)
	}
}

func TestDos20BPBFromBytes(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		bpb, err := Dos20BPBFromBytes(makeValidDos20Bytes())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if bpb.BytesPerSector != SectorSize512 {
			t.Errorf("BytesPerSector = %d, want 512", bpb.BytesPerSector)
		}
		if bpb.RootDirectoryEntries != 224 {
			t.Errorf("RootDirectoryEntries = %d, want 224", bpb.RootDirectoryEntries)
		}
		if bpb.SectorsPerFat != 9 {
			t.Errorf("SectorsPerFat = %d, want 9", bpb.SectorsPerFat)
		}
	})

	t.Run("wrong length", func(t *testing.T) {
		_, err := Dos20BPBFromBytes(make([]byte, 10))
		if err == nil {
			t.Error("expected error for wrong-length input")
		}
	})

	t.Run("invalid sector size", func(t *testing.T) {
		b := makeValidDos20Bytes()
		b[0], b[1] = 0xFF, 0x01 // 511 — not a power of 2
		_, err := Dos20BPBFromBytes(b)
		if err == nil {
			t.Error("expected error for invalid sector size 511")
		}
	})
}

func TestDos20BPBRoundTrip(t *testing.T) {
	original := makeValidDos20Bytes()
	bpb, err := Dos20BPBFromBytes(original)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	got := bpb.ToBytes()
	for i, b := range original {
		if got[i] != b {
			t.Errorf("byte[%d]: got 0x%02X, want 0x%02X", i, got[i], b)
		}
	}
}

func TestDos331BPBFromBytes(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		bpb, err := Dos331BPBFromBytes(makeValidDos331Bytes())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if bpb.SectorsPerTrack != 18 {
			t.Errorf("SectorsPerTrack = %d, want 18", bpb.SectorsPerTrack)
		}
		if bpb.Heads != 2 {
			t.Errorf("Heads = %d, want 2", bpb.Heads)
		}
	})

	t.Run("wrong length", func(t *testing.T) {
		_, err := Dos331BPBFromBytes(make([]byte, 20))
		if err == nil {
			t.Error("expected error for wrong-length input")
		}
	})
}
