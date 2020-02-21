package main

import (
	"fmt"
	"os"

	"github.com/giantswarm/microerror"
	flag "github.com/spf13/pflag"

	"github.com/giantswarm/aws-attach-etcd-dep/aws"
	"github.com/giantswarm/aws-attach-etcd-dep/disk"
	"github.com/giantswarm/aws-attach-etcd-dep/metadata"
	"github.com/giantswarm/aws-attach-etcd-dep/pkg/project"
)

type Flag struct {
	EniDeviceIndex     int64
	EniForceDetach     bool
	EniTagKey          string
	EniTagValue        string
	VolumeDeviceName   string
	VolumeDeviceFsType string
	VolumeForceDetach  bool
	VolumeTagKey       string
	VolumeTagValue     string
}

func main() {
	err := mainError()
	if err != nil {
		panic(fmt.Sprintf("%#v", err))
	}
}

func mainError() error {
	var err error

	var f Flag
	flag.Int64Var(&f.EniDeviceIndex, "eni-device-index", 1, "NIC Device index that will be used for attaching the ENI. Cannot be zeroas that is the default NCI that is already attached.")
	flag.BoolVar(&f.EniForceDetach, "eni-force-detach", false, "If set to true, app will use force-detach if the ENI cannot be detached by normal detach operation..")
	flag.StringVar(&f.EniTagKey, "eni-tag-key", "aws-attach-by-id", "Tag key that will be used to found the requested ENI in AWS API.")
	flag.StringVar(&f.EniTagValue, "eni-tag-value", "test", "Tag value that will be used to found the requested ENI in AWS API, this tag should identify one unique ENI.")

	flag.StringVar(&f.VolumeDeviceName, "volume-device-name", "/dev/xvdh", "Volume device name that will be used for attaching the EBS volume.")
	flag.StringVar(&f.VolumeDeviceFsType, "volume-device-filesystem-type", "ext4", "In case that the EBS device has no file-system, it will be formatted using this value.")
	flag.BoolVar(&f.VolumeForceDetach, "volume-force-detach", false, "If set to true, app will use force-detach if the EBS cannot be detached by normal detach operation.")
	flag.StringVar(&f.VolumeTagKey, "volume-tag-key", "aws-attach-by-id", "Tag key that will be used to found the requested EBS in AWS API.")
	flag.StringVar(&f.VolumeTagValue, "volume-tag-value", "test", "Tag value that will be used to found the requested EBS in AWS API, this tag should identify one unique EBS.")

	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("%s:%s - %s", project.Name(), project.Version(), project.GitSHA())
		return nil
	}
	if len(os.Args) > 1 && os.Args[1] == "--help" {
		flag.Usage()
		return nil
	}
	flag.Parse()

	awsSession, err := getAWSSession()
	if err != nil {
		return microerror.Mask(err)
	}

	instanceID, err := metadata.GetInstanceID(awsSession)
	if err != nil {
		return microerror.Mask(err)
	}
	// attach ENI here
	var eni *aws.ENI
	{
		eniConfig := aws.ENIConfig{
			AWSInstanceID: instanceID,
			AwsSession:    awsSession,
			DeviceIndex:   f.EniDeviceIndex,
			ForceDetach:   f.EniForceDetach,
			TagKey:        f.EniTagKey,
			TagValue:      f.EniTagValue,
		}

		eni, err = aws.NewENI(eniConfig)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	err = eni.AttachByTag()

	if err != nil {
		return microerror.Mask(err)
	}
	// attach EBS here
	// TODO will be added in separate PR

	// it takes a second or two until kernel register the device under `/dev/xxxx`
	err = disk.WaitForDeviceReady(f.VolumeDeviceName)
	if err != nil {
		return microerror.Mask(err)
	}
	err = disk.EnsureDiskHasFileSystem(f.VolumeDeviceName, f.VolumeDeviceFsType)
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}
