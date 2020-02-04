package main

import (
	"fmt"
	"github.com/giantswarm/aws-attach-ebs-by-tag/eni"
	"github.com/giantswarm/aws-attach-ebs-by-tag/metadata"
	"os"

	"github.com/giantswarm/microerror"
	flag "github.com/spf13/pflag"

	"github.com/giantswarm/aws-attach-ebs-by-tag/pkg/project"
	"github.com/giantswarm/aws-attach-ebs-by-tag/volume"
)

type Flag struct {
	EniDeviceIndex    int64
	EniForceDetach    bool
	EniTagKey         string
	EniTagValue       string
	VolumeDeviceName  string
	VolumeForceDetach bool
	VolumeTagKey      string
	VolumeTagValue    string
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
	flag.Int64Var(&f.EniDeviceIndex, "eni-device-name", 1, "NIC Device index that will be used for attaching the ENI. Cannot be zeroas that is the default NCI that is already attached.")
	flag.BoolVar(&f.EniForceDetach, "eni-force-detach", false, "If set to true, app will use force-detach if the ENI cannot be detached by normal detach operation..")
	flag.StringVar(&f.EniTagKey, "eni-tag-key", "aws-attach-by-id", "Tag key that will be used to found the requested ENI in AWS API.")
	flag.StringVar(&f.EniTagValue, "eni-tag-value", "test", "Tag value that will be used to found the requested ENI in AWS API, this tag should identify one unique ENI.")

	flag.StringVar(&f.VolumeDeviceName, "volume-device-name", "/dev/xvdh", "Volume device name that will be used for attaching the EBS volume.")
	flag.BoolVar(&f.VolumeForceDetach, "volume-force-detach", false, "If set to true, app will use force-detach if the EBS cannot be detached by normal detach operation..")
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
	var eniService *eni.Service
	{
		eniConfig := eni.Config{
			AWSInstanceID: instanceID,
			AwsSession:    awsSession,
			DeviceIndex:   f.EniDeviceIndex,
			ForceDetach:   f.EniForceDetach,
			TagKey:        f.EniTagKey,
			TagValue:      f.EniTagValue,
		}

		eniService, err = eni.New(eniConfig)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	err = eniService.AttachEniByTag()

	if err != nil {
		return microerror.Mask(err)
	}

	// attach EBS here
	var volumeService *volume.Service
	{
		attachConfig := volume.Config{
			AWSInstanceID: instanceID,
			AwsSession:    awsSession,
			DeviceName:    f.VolumeDeviceName,
			ForceDetach:   f.VolumeForceDetach,
			TagKey:        f.VolumeTagKey,
			TagValue:      f.VolumeTagValue,
		}

		volumeService, err = volume.New(attachConfig)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	err = volumeService.AttachEBSByTag()

	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}
