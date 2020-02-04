package disk

import (
	"github.com/giantswarm/microerror"
	diskfs "github.com/shirou/gopsutil/disk"
)

func MaybeCreateDiskFileSystem(diskName string, fsType string) error {

	partList, err := diskfs.Partitions(false)
	if err != nil {
		return microerror.Mask(err)
	}

	var diskStat diskfs.PartitionStat

	for _, p := range partList {
		if p.Device == diskName {
			diskStat = p
			break
		}
	}

	if diskStat.Fstype != fsType {

	}
	//diskfs.

	return nil
}

func MountDisk(diskName string, mountDir string) error {

	return nil
}
