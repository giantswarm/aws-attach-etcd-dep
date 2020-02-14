package volume

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/giantswarm/microerror"
)

type Config struct {
	AWSInstanceID string
	AwsSession    *session.Session
	DeviceName    string
	ForceDetach   bool
	TagKey        string
	TagValue      string
}

type Service struct {
	awsInstanceID string
	awsSession    *session.Session
	deviceName    string
	forceDetach   bool
	tagKey        string
	tagValue      string
}

func New(config Config) (*Service, error) {
	if config.AWSInstanceID == "" {
		return nil, microerror.Maskf(invalidConfigError, "config.AWSInstanceID must not be empty")
	}
	if config.AwsSession == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.AwsSession must not be nil")
	}
	if config.DeviceName == "" {
		return nil, microerror.Maskf(invalidConfigError, "config.DeviceName must not be empty")
	}
	if config.TagKey == "" {
		return nil, microerror.Maskf(invalidConfigError, "config.TagKey must not be empty")
	}
	if config.TagValue == "" {
		return nil, microerror.Maskf(invalidConfigError, "config.TagValue must not be empty")
	}

	newService := &Service{
		awsInstanceID: config.AWSInstanceID,
		awsSession:    config.AwsSession,
		deviceName:    config.DeviceName,
		forceDetach:   config.ForceDetach,
		tagKey:        config.TagKey,
		tagValue:      config.TagValue,
	}
	return newService, nil
}

func (s *Service) AttachEBSByTag() error {
	// create ec2 client
	ec2Client := ec2.New(s.awsSession)

	volume, err := s.describeEBSVolume(ec2Client)
	if err != nil {
		return microerror.Mask(err)
	}
	fmt.Printf("Fetched volume-id '%s'\n", *volume.VolumeId)

	if *volume.State == ec2.VolumeStateInUse &&
		len(volume.Attachments) == 1 &&
		*volume.Attachments[0].InstanceId == s.awsInstanceID {
		fmt.Printf("Volume is already attached to this instance. Nothing to do.\n")
		return nil
	} else if *volume.State == ec2.VolumeStateInUse {
		fmt.Printf("Volume is attached to '%s' and is in state '%s'. Trying detach the volume\n", *volume.Attachments[0].InstanceId, *volume.State)

		err := s.detachEBSVolume(ec2Client, volume)
		if err != nil {
			return microerror.Mask(err)
		}
	} else {
		fmt.Printf("Volume state is '%s'.\n", *volume.State)
	}

	err = s.attachEBSVolume(ec2Client, s.awsInstanceID, *volume.VolumeId)
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}
