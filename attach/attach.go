package attach

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/giantswarm/microerror"
)

type Config struct {
	DeviceName  string
	ForceDetach bool
	TagKey      string
	TagValue    string
}

type Service struct {
	deviceName  string
	forceDetach bool
	tagKey      string
	tagValue    string
}

func New(config Config) (*Service, error) {
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
		deviceName:  config.DeviceName,
		forceDetach: config.ForceDetach,
		tagKey:      config.TagKey,
		tagValue:    config.TagValue,
	}
	return newService, nil
}

func (s *Service) AttachEBSByTag() error {
	awsSession := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	region, err := getRegion(awsSession)
	if err != nil {
		return microerror.Mask(err)
	}
	fmt.Printf("Fetched region '%s'\n", region)

	instanceID, err := getInstanceID(awsSession)
	if err != nil {
		return microerror.Mask(err)
	}
	fmt.Printf("Fetched instance-id '%s'\n", instanceID)

	// recreate aws session, this time with proper region
	awsSession, err = session.NewSession(&aws.Config{
		Region: aws.String(region)},
	)
	if err != nil {
		return microerror.Mask(err)
	}
	// create ec2 client
	ec2Client := ec2.New(awsSession)

	volume, err := s.describeEBSVolume(ec2Client)
	if err != nil {
		return microerror.Mask(err)
	}
	fmt.Printf("Fetched volume-id '%s'\n", *volume.VolumeId)

	if *volume.State == ec2.VolumeStateInUse &&
		len(volume.Attachments) == 1 &&
		*volume.Attachments[0].InstanceId == instanceID {
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

	err = s.attachEBSVolume(ec2Client, instanceID, *volume.VolumeId)
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}
