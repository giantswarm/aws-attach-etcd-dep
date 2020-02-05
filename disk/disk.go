package disk

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/giantswarm/microerror"
)

const (
	diskLabel = "var-lib-etcd"
)

var supportedFsType = []string{"ext4"}

func MaybeCreateDiskFileSystem(deviceName string, desiredFsType string) error {
	deviceFsType, err := getFsType(deviceName)
	if err != nil {
		return microerror.Mask(err)
	}
	if deviceFsType == "" {
		// format disk
		err = runMkfs(deviceName, deviceFsType)
		if err != nil {
			return microerror.Mask(err)
		}
	} else {
		fmt.Printf("Block device '%s' has already file-system '%s'.\n", deviceName, desiredFsType)
	}
	return nil
}

func getFsType(deviceName string) (string, error) {
	var out bytes.Buffer
	cmd := exec.Command("/bin/lsblk", "-n", "-o", "FSTYPE", "-f", deviceName)
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", microerror.Maskf(err, "failed to check fs type")
	}
	return strings.TrimSpace(out.String()), nil
}

func runMkfs(deviceName string, fsType string) error {
	supported := false
	for _, i := range supportedFsType {
		if i == fsType {
			supported = true
			break
		}
	}
	if !supported {
		return microerror.Maskf(executionFailedError, fmt.Sprintf("fsType '%s' is not supported", fsType))
	}

	cmd := exec.Command("/sbin/mkfs", "-t", fsType, "L", diskLabel, deviceName)
	err := cmd.Run()

	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}
