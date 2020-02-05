package eni

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/giantswarm/microerror"
)

type Config struct {
	AWSInstanceID string
	AwsSession    *session.Session
	DeviceIndex   int64
	ForceDetach   bool
	TagKey        string
	TagValue      string
}

type Service struct {
	awsInstanceID string
	awsSession    *session.Session
	deviceIndex   int64
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
	if config.DeviceIndex == 0 {
		return nil, microerror.Maskf(invalidConfigError, "config.DeviceIndex must not be 0")
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
		deviceIndex:   config.DeviceIndex,
		forceDetach:   config.ForceDetach,
		tagKey:        config.TagKey,
		tagValue:      config.TagValue,
	}
	return newService, nil
}

func (s *Service) AttachEniByTag() error {
	// create ec2 client
	ec2Client := ec2.New(s.awsSession)

	eni, err := s.describeEni(ec2Client)
	if err != nil {
		return microerror.Mask(err)
	}
	fmt.Printf("Fetched eni-id '%s'\n", *eni.NetworkInterfaceId)

	if *eni.Status == ec2.NetworkInterfaceStatusInUse &&
		*eni.Attachment.InstanceId == s.awsInstanceID {
		fmt.Printf("ENI is already attached to this instance. Nothing to do.\n")
		return nil
	} else if *eni.Status == ec2.NetworkInterfaceStatusInUse {
		fmt.Printf("ENI is attached to '%s' and is in state '%s'. Trying detach the volume\n", *eni.Attachment.InstanceId, *eni.Status)

		err := s.detachEni(ec2Client, eni)
		if err != nil {
			return microerror.Mask(err)
		}
	} else {
		fmt.Printf("ENI state is '%s'.\n", *eni.Status)
	}

	err = s.attachEni(ec2Client, s.awsInstanceID, *eni.NetworkInterfaceId)
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}
