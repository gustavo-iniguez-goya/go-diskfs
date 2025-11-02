package partition

import (
	"github.com/gustavo-iniguez-goya/go-diskfs/backend"
	"github.com/gustavo-iniguez-goya/go-diskfs/partition/part"
)

// Table reference to a partitioning table on disk
type Table interface {
	Type() string
	Write(backend.WritableFile, int64) error
	GetPartitions() []part.Partition
	Repair(diskSize uint64) error
	Verify(f backend.File, diskSize uint64) error
	UUID() string
}
