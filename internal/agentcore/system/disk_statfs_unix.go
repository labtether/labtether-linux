//go:build linux || darwin || freebsd

package system

import (
	"syscall"

	"github.com/labtether/labtether-linux/pkg/agentmgr"
)

// StatfsMountPoint uses syscall.Statfs to collect disk space info for a mount point.
func StatfsMountPoint(device, mountPoint, fsType string) (agentmgr.MountInfo, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(mountPoint, &stat); err != nil {
		return agentmgr.MountInfo{}, err
	}

	blockSize := uint64(0)
	if stat.Bsize > 0 {
		blockSize = uint64(stat.Bsize)
	}

	// FreeBSD exposes Bavail as a signed value because reserved filesystem
	// blocks can make it negative. Guard before converting or a small negative
	// value becomes a huge capacity.
	availableBlocks := uint64(0)
	if stat.Bavail > 0 {
		availableBlocks = uint64(stat.Bavail)
	}

	total := saturatingBlockBytes(uint64(stat.Blocks), blockSize)
	available := saturatingBlockBytes(availableBlocks, blockSize)
	free := saturatingBlockBytes(uint64(stat.Bfree), blockSize)
	used := uint64(0)
	if total > free {
		used = total - free
	}

	var usePct float64
	if total > 0 {
		usePct = float64(used) / float64(total) * 100
	}

	return agentmgr.MountInfo{
		Device:     device,
		MountPoint: mountPoint,
		FSType:     fsType,
		Total:      total,
		Used:       used,
		Available:  available,
		UsePct:     usePct,
	}, nil
}

func saturatingBlockBytes(blocks, blockSize uint64) uint64 {
	if blocks == 0 || blockSize == 0 {
		return 0
	}
	maxUint64 := ^uint64(0)
	if blocks > maxUint64/blockSize {
		return maxUint64
	}
	return blocks * blockSize
}
