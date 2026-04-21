package fat12

// table_internal_test.go exercises the FAT12 12-bit packing logic and the
// fat16TableImpl (16-bit) using white-box access to unexported helpers.

import (
	"testing"
)

// ── 12-bit packing helpers ────────────────────────────────────────────────────

// knownGoodFAT12Bytes encodes the following cluster chain in a minimal FAT12 byte
// stream (entries 0-5 packed into 9 bytes):
//
//	entry 0 = 0xFF8  (FAT ID / media descriptor)
//	entry 1 = 0xFFF  (EOC for entry 1)
//	entry 2 = 0xFFF  (EOC — root dir)
//	entry 3 = 0x004  (cluster 3 → 4)
//	entry 4 = 0xFFF  (EOC)
//	entry 5 = 0x000  (free)
//
// Manual derivation (3 bytes per 2 entries):
//
//	bytes 0-2 encode entries 0-1:
//	  entry0=0xFF8 → b[0]=0xF8, b[1]=(b[1]&0xF0)|0x0F → b[1] upper nibble from entry1
//	  entry1=0xFFF → b[1]=(0x0F&0x0F)|(0xFF<<4&0xF0)=0xFF, b[2]=0xFF>>4... wait
//
// Easier to compute via fat12WriteEntry and check the result byte-by-byte.
func TestFat12ReadEntry(t *testing.T) {
	// 9 bytes encode 6 entries (entries 0-5).
	// We construct them by writing known values and then reading them back.
	b := make([]byte, 9)

	values := []uint32{0xFF8, 0xFFF, 0xFFF, 0x004, 0xFFF, 0x000}
	for i, v := range values {
		fat12WriteEntry(b, uint32(i), v)
	}

	for i, want := range values {
		got := fat12ReadEntry(b, uint32(i))
		if got != want {
			t.Errorf("entry %d: got 0x%03X, want 0x%03X (bytes: %X)", i, got, want, b)
		}
	}
}

func TestFat12WriteReadRoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		values []uint32
	}{
		{"all zeros", []uint32{0, 0, 0, 0}},
		{"all EOC", []uint32{0xFFF, 0xFFF, 0xFFF, 0xFFF}},
		{"chain 2→3→EOC, 4 free", []uint32{0xFF8, 0xFFF, 0x003, 0xFFF, 0x000}},
		{"alternating even/odd", []uint32{0x123, 0x456, 0x789, 0xABC, 0xDEF}},
		{"max values", []uint32{0xFFF, 0xFFE, 0xFFD, 0xFFC, 0xFFB}},
		{"boundary nibbles", []uint32{0x001, 0x100, 0x010, 0xF0F, 0x0F0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := uint32(len(tt.values)*3/2 + 1)
			b := make([]byte, size)
			for i, v := range tt.values {
				fat12WriteEntry(b, uint32(i), v)
			}
			for i, want := range tt.values {
				got := fat12ReadEntry(b, uint32(i))
				if got != want {
					t.Errorf("entry %d: round-trip got 0x%03X, want 0x%03X", i, got, want)
				}
			}
		})
	}
}

// TestFat12ThreeByteSharing verifies the key invariant: two adjacent entries
// share exactly 3 bytes and neither corrupts the other.
func TestFat12ThreeByteSharing(t *testing.T) {
	// Write entry 0 then entry 1; confirm that writing one does not clobber the other.
	tests := []struct{ e0, e1 uint32 }{
		{0x000, 0x000},
		{0xFFF, 0x000},
		{0x000, 0xFFF},
		{0xFFF, 0xFFF},
		{0xABC, 0xDEF},
		{0x001, 0xFFE},
		{0xFFE, 0x001},
		{0x800, 0x7FF},
	}
	for _, tt := range tests {
		b := make([]byte, 3)
		fat12WriteEntry(b, 0, tt.e0)
		fat12WriteEntry(b, 1, tt.e1)
		got0 := fat12ReadEntry(b, 0)
		got1 := fat12ReadEntry(b, 1)
		if got0 != tt.e0 || got1 != tt.e1 {
			t.Errorf("e0=0x%03X e1=0x%03X: round-trip got e0=0x%03X e1=0x%03X (bytes=%X)",
				tt.e0, tt.e1, got0, got1, b)
		}
	}
}

// ── fat12Table ────────────────────────────────────────────────────────────────

func TestFat12TableIsEOC(t *testing.T) {
	tbl := newFat12Table(0x0FF8, 512)
	tests := []struct {
		val  uint32
		want bool
	}{
		{0x000, false},
		{0x001, false},
		{0xFF6, false},
		{0xFF7, false}, // bad-cluster mark, not EOC
		{0xFF8, true},
		{0xFF9, true},
		{0xFFA, true},
		{0xFFB, true},
		{0xFFC, true},
		{0xFFD, true},
		{0xFFE, true},
		{0xFFF, true},
	}
	for _, tt := range tests {
		if got := tbl.IsEOC(tt.val); got != tt.want {
			t.Errorf("IsEOC(0x%03X) = %v, want %v", tt.val, got, tt.want)
		}
	}
}

func TestFat12TableBytesRoundTrip(t *testing.T) {
	// Construct a table, serialise, re-parse, compare.
	sizeBytes := uint32(512)
	tbl := newFat12Table(0x0FF8, sizeBytes)
	// Mark some clusters.
	tbl.clusters[2] = tbl.eoc // root dir
	tbl.clusters[3] = 4
	tbl.clusters[4] = 5
	tbl.clusters[5] = tbl.eoc
	tbl.clusters[6] = tbl.eoc

	b := tbl.Bytes()
	if uint32(len(b)) != sizeBytes {
		t.Fatalf("Bytes() length = %d, want %d", len(b), sizeBytes)
	}

	tbl2 := newFat12Table(0, sizeBytes)
	tbl2.FromBytes(b)

	for i := uint32(2); i <= tbl.max; i++ {
		if tbl2.clusters[i] != tbl.clusters[i] {
			t.Errorf("cluster[%d]: got 0x%03X, want 0x%03X", i, tbl2.clusters[i], tbl.clusters[i])
		}
	}
}

func TestFat12TableSetGetCluster(t *testing.T) {
	tbl := newFat12Table(0x0FF8, 512)

	tbl.SetCluster(2, 0xFFF)
	if got := tbl.ClusterValue(2); got != 0xFFF {
		t.Errorf("ClusterValue(2) = 0x%03X, want 0xFFF", got)
	}

	tbl.SetCluster(3, 4)
	tbl.SetCluster(4, tbl.EOCMarker())
	if got := tbl.ClusterValue(3); got != 4 {
		t.Errorf("ClusterValue(3) = %d, want 4", got)
	}
	if !tbl.IsEOC(tbl.ClusterValue(4)) {
		t.Errorf("ClusterValue(4) should be EOC, got 0x%03X", tbl.ClusterValue(4))
	}
}

func TestFat12TableMarkers(t *testing.T) {
	tbl := newFat12Table(0x0FF8, 512)
	if tbl.UnusedMarker() != 0 {
		t.Errorf("UnusedMarker() = %d, want 0", tbl.UnusedMarker())
	}
	if !tbl.IsEOC(tbl.EOCMarker()) {
		t.Errorf("EOCMarker() should satisfy IsEOC")
	}
	if tbl.RootDirCluster() != 2 {
		t.Errorf("RootDirCluster() = %d, want 2", tbl.RootDirCluster())
	}
}
