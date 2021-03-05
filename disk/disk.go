package disk

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"
)

const (
	maxRetries    = 15
	retryInterval = time.Second * 10
)

var supportedFsType = []string{"ext4"}

func WaitForDeviceReady(deviceName string) error {
	b := backoff.NewMaxRetries(maxRetries, retryInterval)
	o := func() error {
		_, err := os.Stat(deviceName)
		if os.IsNotExist(err) {
			fmt.Printf("Waiting until device %q is registered by kernel.\n", deviceName)
			return err
		}
		return nil
	}
	err := backoff.Retry(o, b)
	if err != nil {
		fmt.Printf("wait limit exceeded for device %q after %d retries\n", deviceName, maxRetries)
		return microerror.Mask(err)
	}
	return nil
}

func EnsureDiskHasFileSystem(deviceName string, desiredFsType string, desiredLabel string) error {
	deviceFsType, err := getFsType(deviceName)
	if err != nil {
		return microerror.Mask(err)
	}
	if deviceFsType == "" {
		// format disk
		err = runMkfs(deviceName, desiredFsType, desiredLabel)
		if err != nil {
			return microerror.Mask(err)
		}
	} else if deviceFsType != desiredFsType {
		return microerror.Maskf(executionFailedError, fmt.Sprintf("Block device has unexpected fs type %q.", deviceFsType))
	} else {
		fmt.Printf("Block device %q has already file-system %q.\n", deviceName, desiredFsType)
	}
	return nil
}

func MountDisk(deviceName string, mountPath string, fsType string) error {
	var out, outError bytes.Buffer
	cmd := exec.Command("/bin/mount", "-t", fsType, deviceName, mountPath)
	cmd.Stdout = &out
	cmd.Stderr = &outError
	err := cmd.Run()
	if err != nil {
		return microerror.Maskf(err, fmt.Sprintf("failed to mount disk '%s', err: %s", deviceName, outError.String()))
	}
	fmt.Printf("The device '%s' was mounted to '%s'\n", deviceName, mountPath)
	return nil
}

func getFsType(deviceName string) (string, error) {
	var out, outError bytes.Buffer
	cmd := exec.Command("/bin/lsblk", "-n", "-o", "FSTYPE", "-f", deviceName)
	cmd.Stdout = &out
	cmd.Stderr = &outError
	err := cmd.Run()
	if err != nil {
		return "", microerror.Maskf(err, fmt.Sprintf("failed to check fs type for '%s', err: %s", deviceName, outError.String()))
	}
	return strings.TrimSpace(out.String()), nil
}

func runMkfs(deviceName string, fsType string, label string) error {
	supported := false
	for _, i := range supportedFsType {
		if i == fsType {
			supported = true
			break
		}
	}
	if !supported {
		return microerror.Maskf(executionFailedError, fmt.Sprintf("fsType %q is not supported", fsType))
	}

	cmd := exec.Command("/sbin/mkfs", "-t", fsType, "-L", label, deviceName)
	err := cmd.Run()

	if err != nil {
		return microerror.Mask(err)
	}
	fmt.Printf("Device '%s' was formatted as '%s'\n", deviceName, fsType)
	return nil
}
