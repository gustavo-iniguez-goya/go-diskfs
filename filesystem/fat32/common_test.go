package fat32

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const (
	Fat32File    = "./testdata/dist/fat32.img"
	Fat32File4kB = "./testdata/dist/fat32-4k.img"
	fsckFile     = "./testdata/dist/fsck.txt"
	rootdirFile  = "./testdata/dist/root_dir.txt"
	serialFile   = "./testdata/dist/serial.txt"
	fsstatFile   = "./testdata/dist/fsstat.txt"
)

type testFSInfo struct {
	bytesPerCluster uint32
	dataStartBytes  uint32
	dataStartSector uint32
	bytesPerSector  uint32
	reservedSectors uint32
	sectorsPerFAT   uint32
	label           string
	serial          uint32
	sectorsPerTrack uint32
	heads           uint32
	hiddenSectors   uint32
	freeSectorCount uint32
	nextFreeSector  uint32
	firstFAT        uint32
	table           *table
}

var (
	testVolumeLabelRE       = regexp.MustCompile(`^\s*Volume in drive\s+:\s+is\s+(.+)\s*$`)
	testFSCKDataStart       = regexp.MustCompile(`Data area starts at byte (\d+) \(sector (\d+)\)`)
	testFSCKBytesPerSector  = regexp.MustCompile(`^\s*(\d+) bytes per logical sector\s*$`)
	testFSCKBytesPerCluster = regexp.MustCompile(`^\s*(\d+) bytes per cluster\s*$`)
	testFSCKReservedSectors = regexp.MustCompile(`^\s*(\d+) reserved sectors\s*$`)
	testFSCKSectorsPerFat   = regexp.MustCompile(`^\s*(\d+) bytes per FAT \(= (\d+) sectors\)\s*$`)
	testFSCKHeadsSectors    = regexp.MustCompile(`^\s*(\d+) sectors/track, (\d+) heads\s*$`)
	testFSCKHiddenSectors   = regexp.MustCompile(`^\s*(\d+) hidden sectors\s*$`)
	testFSCKFirstFAT        = regexp.MustCompile(`^\s*First FAT starts at byte (\d+) \(sector (\d+)\)\s*$`)

	testFSSTATFreeSectorCountRE = regexp.MustCompile(`^\s*Free Sector Count.*: (\d+)\s*$`)
	testFSSTATNextFreeSectorRE  = regexp.MustCompile(`^\s*Next Free Sector.*: (\d+)\s*`)
	testFSSTATClustersStartRE   = regexp.MustCompile(`\s*FAT CONTENTS \(in sectors\)\s*$`)
	testFSSTATClusterLineRE     = regexp.MustCompile(`\s*(\d+)-(\d+) \((\d+)\)\s+->\s+(\S+)\s*$`)

	fsInfo *testFSInfo
)

// TestMain sets up the test environment and runs the tests.
func TestMain(m *testing.M) {
	if _, err := os.Stat(Fat32File); os.IsNotExist(err) {
		cmd := exec.Command("sh", "mkfat32.sh")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = "testdata"
		if err := cmd.Run(); err != nil {
			println("error generating test artifacts for fat32", err)
			os.Exit(1)
		}
	}

	var err error
	fsInfo, err = testReadFilesystemData()
	if err != nil {
		println("Error reading fsck file", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

//nolint:gocyclo // parsing multiple fsck/fsstat output formats requires many branches
func testReadFilesystemData() (info *testFSInfo, err error) {
	info = &testFSInfo{}
	fsckInfo, err := os.ReadFile(fsckFile)
	if err != nil {
		return nil, fmt.Errorf("error opening fsck info file %s: %v", fsckFile, err)
	}
	scanner := bufio.NewScanner(bytes.NewReader(fsckInfo))
	for scanner.Scan() {
		text := scanner.Text()
		dataStartMatch := testFSCKDataStart.FindStringSubmatch(text)
		bytesPerClusterMatch := testFSCKBytesPerCluster.FindStringSubmatch(text)
		bytesPerSectorMatch := testFSCKBytesPerSector.FindStringSubmatch(text)
		reservedSectorsMatch := testFSCKReservedSectors.FindStringSubmatch(text)
		sectorsPerFATMatch := testFSCKSectorsPerFat.FindStringSubmatch(text)
		headsSectorMatch := testFSCKHeadsSectors.FindStringSubmatch(text)
		hiddenSectorsMatch := testFSCKHiddenSectors.FindStringSubmatch(text)
		firstFATMatch := testFSCKFirstFAT.FindStringSubmatch(text)
		switch {
		case len(headsSectorMatch) == 3:
			sectorsPerTrack, err := strconv.Atoi(headsSectorMatch[1])
			if err != nil {
				return nil, fmt.Errorf("error parsing sectors per track %s: %v", headsSectorMatch[1], err)
			}
			heads, err := strconv.Atoi(headsSectorMatch[2])
			if err != nil {
				return nil, fmt.Errorf("error parsing heads %s: %v", headsSectorMatch[2], err)
			}
			info.sectorsPerTrack = uint32(sectorsPerTrack)
			info.heads = uint32(heads)
		case len(hiddenSectorsMatch) == 2:
			hiddenSectors, err := strconv.Atoi(hiddenSectorsMatch[1])
			if err != nil {
				return nil, fmt.Errorf("error parsing hidden sectors %s: %v", hiddenSectorsMatch[1], err)
			}
			info.hiddenSectors = uint32(hiddenSectors)
		case len(dataStartMatch) == 3:
			byteStart, err := strconv.Atoi(dataStartMatch[1])
			if err != nil {
				return nil, fmt.Errorf("error parsing data start byte %s: %v", dataStartMatch[1], err)
			}
			sectorStart, err := strconv.Atoi(dataStartMatch[2])
			if err != nil {
				return nil, fmt.Errorf("error parsing data start sector %s: %v", dataStartMatch[2], err)
			}
			info.dataStartBytes = uint32(byteStart)
			info.dataStartSector = uint32(sectorStart)
		case len(bytesPerClusterMatch) == 2:
			bytesPerCluster, err := strconv.Atoi(bytesPerClusterMatch[1])
			if err != nil {
				return nil, fmt.Errorf("error parsing bytes per cluster %s: %v", bytesPerClusterMatch[1], err)
			}
			info.bytesPerCluster = uint32(bytesPerCluster)
		case len(bytesPerSectorMatch) == 2:
			bytesPerSector, err := strconv.Atoi(bytesPerSectorMatch[1])
			if err != nil {
				return nil, fmt.Errorf("error parsing bytes per sector %s: %v", bytesPerSectorMatch[1], err)
			}
			info.bytesPerSector = uint32(bytesPerSector)
		case len(reservedSectorsMatch) == 2:
			reservedSectors, err := strconv.Atoi(reservedSectorsMatch[1])
			if err != nil {
				return nil, fmt.Errorf("error parsing reserved sectors %s: %v", reservedSectorsMatch[1], err)
			}
			info.reservedSectors = uint32(reservedSectors)
		case len(sectorsPerFATMatch) == 3:
			sectorsPerFAT, err := strconv.Atoi(sectorsPerFATMatch[2])
			if err != nil {
				return nil, fmt.Errorf("error parsing sectors per FAT %s: %v", sectorsPerFATMatch[2], err)
			}
			info.sectorsPerFAT = uint32(sectorsPerFAT)
		case len(firstFATMatch) == 3:
			firstFAT, err := strconv.Atoi(firstFATMatch[1])
			if err != nil {
				return nil, fmt.Errorf("error parsing first FAT byte %s: %v", firstFATMatch[1], err)
			}
			info.firstFAT = uint32(firstFAT)
		}
	}

	dirInfo, err := os.ReadFile(rootdirFile)
	if err != nil {
		println("Error opening directory info file", rootdirFile, err)
		os.Exit(1)
	}
	scanner = bufio.NewScanner(bytes.NewReader(dirInfo))
	for scanner.Scan() {
		text := scanner.Text()
		volLabelMatch := testVolumeLabelRE.FindStringSubmatch(text)
		if len(volLabelMatch) == 2 {
			info.label = strings.TrimSpace(volLabelMatch[1])
			break
		}
	}

	serial, err := os.ReadFile(serialFile)
	if err != nil {
		println("Error reading serial file", serialFile, err)
		os.Exit(1)
	}
	decimal, err := strconv.ParseInt(strings.TrimSpace(string(serial)), 16, 64)
	if err != nil {
		println("Error converting contents of serial file to integer:", err)
		os.Exit(1)
	}
	info.serial = uint32(decimal)

	fsstat, err := os.ReadFile(fsstatFile)
	if err != nil {
		println("Error reading fsstat file", fsstatFile, err)
		os.Exit(1)
	}
	scanner = bufio.NewScanner(bytes.NewReader(fsstat))
	var inClusters bool
	for scanner.Scan() {
		text := scanner.Text()
		freeSectorsMatch := testFSSTATFreeSectorCountRE.FindStringSubmatch(text)
		nextFreeSectorMatch := testFSSTATNextFreeSectorRE.FindStringSubmatch(text)
		clusterStartMatch := testFSSTATClustersStartRE.FindStringSubmatch(text)
		clusterLineMatch := testFSSTATClusterLineRE.FindStringSubmatch(text)
		switch {
		case len(freeSectorsMatch) == 2:
			freeSectors, err := strconv.Atoi(freeSectorsMatch[1])
			if err != nil {
				println("Error parsing free sectors count", freeSectorsMatch[1], err)
				os.Exit(1)
			}
			info.freeSectorCount = uint32(freeSectors)
		case len(nextFreeSectorMatch) == 2:
			nextFreeSector, err := strconv.Atoi(nextFreeSectorMatch[1])
			if err != nil {
				println("Error parsing next free sector", nextFreeSectorMatch[1], err)
				os.Exit(1)
			}
			info.nextFreeSector = uint32(nextFreeSector) - info.dataStartSector + 2
		case len(clusterStartMatch) > 0:
			inClusters = true
			sectorsPerFat := info.sectorsPerFAT
			sizeInBytes := sectorsPerFat * info.bytesPerSector
			numClusters := sizeInBytes / 4
			info.table = &table{
				fatID:          268435448, // 0x0ffffff8
				eocMarker:      eoc,       // 0x0fffffff
				rootDirCluster: 2,
				size:           sizeInBytes,
				maxCluster:     numClusters,
				clusters:       make([]uint32, numClusters+1),
			}
		case inClusters && len(clusterLineMatch) > 4:
			start, err := strconv.Atoi(clusterLineMatch[1])
			if err != nil {
				println("Error parsing cluster start", clusterLineMatch[1], err)
				os.Exit(1)
			}
			end, err := strconv.Atoi(clusterLineMatch[2])
			if err != nil {
				println("Error parsing cluster end", clusterLineMatch[2], err)
				os.Exit(1)
			}
			var target uint32
			if clusterLineMatch[4] == "EOF" {
				target = eoc
			} else {
				targetInt, err := strconv.Atoi(clusterLineMatch[4])
				if err != nil {
					println("Error parsing cluster target", clusterLineMatch[4], err)
					os.Exit(1)
				}
				target = uint32(targetInt) - info.dataStartSector + 2
			}
			for i := start; i < end; i++ {
				startCluster := uint32(i) - info.dataStartSector + 2
				info.table.clusters[startCluster] = startCluster + 1
			}
			endCluster := uint32(end) - info.dataStartSector + 2
			if endCluster == 2 {
				target = eocMin
			}
			info.table.clusters[endCluster] = target
		}
	}
	return info, err
}
