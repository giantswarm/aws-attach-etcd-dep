package disk

import (
	"fmt"

	"github.com/giantswarm/microerror"
	diskfs "github.com/shirou/gopsutil/disk"
)

const diskLabel = "var-lib-etcd"

func MaybeCreateDiskFileSystem(deviceName string, fsType string) error {

	partList, err := diskfs.Partitions(false)
	if err != nil {
		return microerror.Mask(err)
	}

	var diskStat diskfs.PartitionStat
	for _, p := range partList {
		if p.Device == deviceName {
			diskStat = p
			break
		}
	}
	fmt.Printf("debug: %#v\n", partList)
	if diskStat.Device == "" {
		fmt.Printf("Did not any find any block device '%s'.\n", deviceName)

		return microerror.Maskf(executionFailedError, fmt.Sprintf("block device '%s' not found", deviceName))
	} else {
		fmt.Printf("Found block device '%s' with fs type '%s'", diskStat.Device, diskStat.Fstype)
	}
	if diskStat.Fstype == "" {
		// format disk
		fmt.Printf("TODO run 'mkfs -t %s -L %s %s'\n", fsType, diskLabel, deviceName)
	} else {
		fmt.Printf("Block device '%s' has already file-system '%s'.\n", deviceName, fsType)
	}
	return nil
}
