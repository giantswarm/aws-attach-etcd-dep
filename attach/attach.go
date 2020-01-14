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

	ec2Client := ec2.New(awsSession)

	instanceID, err := getInstanceID(awsSession)
	if err != nil {
		return microerror.Mask(err)
	}
	fmt.Printf("Fetched Instance-ID '%s'\n", instanceID)

	volumeID, err := s.getEBSVolumeID(ec2Client)
	if err != nil {
		return microerror.Mask(err)
	}
	fmt.Printf("Fetched Volume-ID '%s'\n", volumeID)

	attachVolumeInput := &ec2.AttachVolumeInput{
		Device:     aws.String(s.deviceName),
		InstanceId: aws.String(instanceID),
		VolumeId:   aws.String(volumeID),
	}

	attachment, err := ec2Client.AttachVolume(attachVolumeInput)
	if err != nil {
		return microerror.Mask(err)
	}

	fmt.Printf("Succefully attached volume. %s\n", attachment.String())
	return nil
}
