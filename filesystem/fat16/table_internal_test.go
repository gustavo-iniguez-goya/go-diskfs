package fat16

import "testing"

func TestFat16TableIsEOC(t *testing.T) {
	tbl := newFat16Table(0xFFF8, 512)
	tests := []struct {
		val  uint32
		want bool
	}{
		{0x0000, false},
		{0xFFF6, false},
		{0xFFF7, false}, // bad-cluster mark, not EOC
		{0xFFF8, true},
		{0xFFF9, true},
		{0xFFFE, true},
		{0xFFFF, true},
	}
	for _, tt := range tests {
		if got := tbl.IsEOC(tt.val); got != tt.want {
			t.Errorf("IsEOC(0x%04X) = %v, want %v", tt.val, got, tt.want)
		}
	}
}

func TestFat16TableBytesRoundTrip(t *testing.T) {
	sizeBytes := uint32(512)
	tbl := newFat16Table(0xFFF8, sizeBytes)
	tbl.SetCluster(2, tbl.EOCMarker())
	tbl.SetCluster(3, 4)
	tbl.SetCluster(4, tbl.EOCMarker())

	b := tbl.Bytes()
	if uint32(len(b)) != sizeBytes {
		t.Fatalf("Bytes() length = %d, want %d", len(b), sizeBytes)
	}

	tbl2 := newFat16Table(0, sizeBytes)
	tbl2.FromBytes(b)

	if !tbl.equal(tbl2) {
		for i := uint32(2); i < sizeBytes/2; i++ {
			if tbl2.ClusterValue(i) != tbl.ClusterValue(i) {
				t.Errorf("cluster[%d]: got 0x%04X, want 0x%04X",
					i, tbl2.ClusterValue(i), tbl.ClusterValue(i))
			}
		}
	}
}

func TestFat16TableMarkers(t *testing.T) {
	tbl := newFat16Table(0xFFF8, 512)
	if tbl.UnusedMarker() != 0 {
		t.Errorf("UnusedMarker() = %d, want 0", tbl.UnusedMarker())
	}
	if !tbl.IsEOC(tbl.EOCMarker()) {
		t.Error("EOCMarker() should satisfy IsEOC")
	}
	if tbl.RootDirCluster() != 2 {
		t.Errorf("RootDirCluster() = %d, want 2", tbl.RootDirCluster())
	}
}
