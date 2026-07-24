//go:build linux

package sysconfig

import (
	"os"
	"strconv"
	"strings"
)

// ReadIfaceStats reads RX/TX byte and packet counters from the Linux sysfs
// network statistics interface at /sys/class/net/{name}/statistics/.
// Returns zeroes if any file cannot be read.
func ReadIfaceStats(name string) (rxBytes, txBytes, rxPackets, txPackets uint64) {
	if !validLinuxInterfaceName(name) {
		return
	}
	statsRoot, err := os.OpenRoot("/sys/class/net/" + name + "/statistics")
	if err != nil {
		return
	}
	defer statsRoot.Close()

	rxBytes = readUint64File(statsRoot, "rx_bytes")
	txBytes = readUint64File(statsRoot, "tx_bytes")
	rxPackets = readUint64File(statsRoot, "rx_packets")
	txPackets = readUint64File(statsRoot, "tx_packets")
	return
}

// readUint64File reads a single uint64 value from a sysfs file.
func readUint64File(root *os.Root, name string) uint64 {
	data, err := root.ReadFile(name)
	if err != nil {
		return 0
	}
	v, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0
	}
	return v
}

func validLinuxInterfaceName(name string) bool {
	return name != "" &&
		name != "." &&
		name != ".." &&
		!strings.ContainsAny(name, "/\\\x00")
}
