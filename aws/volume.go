package aws

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/aws-attach-etcd-dep/metadata"
)

type EBSConfig struct {
	AWSInstanceID string
	AwsSession    *session.Session
	DeviceName    string
	ForceDetach   bool
	TagKey        string
	TagValue      string
}

type EBS struct {
	awsInstanceID string
	awsSession    *session.Session
	deviceName    string
	forceDetach   bool
	tagKey        string
	tagValue      string
}

func NewEBS(config EBSConfig) (*EBS, error) {
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

	newEBS := &EBS{
		awsInstanceID: config.AWSInstanceID,
		awsSession:    config.AwsSession,
		deviceName:    config.DeviceName,
		forceDetach:   config.ForceDetach,
		tagKey:        config.TagKey,
		tagValue:      config.TagValue,
	}
	return newEBS, nil
}

func (s *EBS) AttachByTag() error {
	// create ec2 client
	ec2Client := ec2.New(s.awsSession)

	volume, err := s.describe(ec2Client)
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
		fmt.Printf("Volume is attached to %q and is in state %q. Trying detach the volume\n", *volume.Attachments[0].InstanceId, *volume.State)

		err := s.detach(ec2Client, volume)
		if err != nil {
			return microerror.Mask(err)
		}
	} else {
		fmt.Printf("Volume state is %q.\n", *volume.State)
	}

	err = s.attach(ec2Client, s.awsInstanceID, *volume.VolumeId)
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}

func (s *EBS) describe(ec2Client *ec2.EC2) (*ec2.Volume, error) {
	az, err := metadata.GetAZ(s.awsSession)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	volumeAZFilter := &ec2.Filter{
		Name:   azKeyFilter(),
		Values: tagValue(az),
	}
	volumeTagFilter := &ec2.Filter{
		Name:   tagKey(s.tagKey),
		Values: tagValue(s.tagValue),
	}

	describeVolumeInput := &ec2.DescribeVolumesInput{
		Filters: []*ec2.Filter{
			volumeAZFilter,
			volumeTagFilter,
		},
	}
	o, err := ec2Client.DescribeVolumes(describeVolumeInput)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// tags should give us only one unique volume
	if len(o.Volumes) != 1 {
		return nil, microerror.Maskf(executionFailedError, "expected 1 volume but got %d instead", len(o.Volumes))
	}

	return o.Volumes[0], nil
}

func (s *EBS) attach(ec2Client *ec2.EC2, instanceID string, volumeID string) error {
	attachVolumeInput := &ec2.AttachVolumeInput{
		Device:     aws.String(s.deviceName),
		InstanceId: aws.String(instanceID),
		VolumeId:   aws.String(volumeID),
	}

	b := backoff.NewMaxRetries(attachRequestRetries, retryInterval)
	o := func() error {
		attachment, err := ec2Client.AttachVolume(attachVolumeInput)
		if err != nil {
			return microerror.Mask(err)
		}

		fmt.Printf("Succefully created attach request. %q\n", attachment.String())
		return nil
	}

	err := backoff.Retry(o, b)
	if err != nil {
		fmt.Printf("Failed to create volume volume attach request after %d retries.\n", attachRequestRetries)
		return microerror.Mask(err)
	}

	b = backoff.NewMaxRetries(maxRetries, retryInterval)
	o = func() error {
		volume, err := s.describe(ec2Client)
		if err != nil {
			return microerror.Mask(err)
		}

		if *volume.State != ec2.VolumeStateInUse && *volume.Attachments[0].InstanceId == instanceID {
			fmt.Printf("Volume state is %q, expecting %q, retrying in %ds.\n", *volume.State, ec2.VolumeStateInUse, retryInterval/time.Second)
			return microerror.Maskf(executionFailedError, "EBS not attached")
		}
		return nil
	}
	err = backoff.Retry(o, b)
	if err != nil {
		fmt.Printf("Failed to attach volume after %d retries.\n", maxRetries)
		return microerror.Mask(err)
	}

	fmt.Printf("Volume attached, state %q.\n", ec2.VolumeStateInUse)
	return nil
}

func (s *EBS) detach(ec2Client *ec2.EC2, volume *ec2.Volume) error {
	// wait if automatic detach happens  by terminating the instance
	b := backoff.NewMaxRetries(waitAutoDetachMaxRetries, retryInterval)
	o := func() error {
		volume, err := s.describe(ec2Client)
		if err != nil {
			return microerror.Mask(err)
		}

		if *volume.State != ec2.VolumeStateAvailable {
			fmt.Printf("Volume state is %q, expecting %q, retrying in %ds.\n", *volume.State, ec2.VolumeStateAvailable, retryInterval/time.Second)
			return microerror.Maskf(executionFailedError, "EBS not detached")
		}
		return nil
	}

	err := backoff.Retry(o, b)
	if err != nil {
		// the Volume was eventually detached by the instance by itself, no need for manual detach
		return nil
	} else {
		// volume is still attached after 10mins, lets try detach it manually here
		detachVolumeInput := &ec2.DetachVolumeInput{
			Device:     volume.Attachments[0].Device,
			InstanceId: volume.Attachments[0].InstanceId,
			VolumeId:   volume.VolumeId,
			Force:      aws.Bool(s.forceDetach),
		}

		detachment, err := ec2Client.DetachVolume(detachVolumeInput)
		if strings.Contains(err.Error(), "IncorrectState") {
			// volume is probably already detached, lets ignore the error
		} else if err != nil {
			return microerror.Mask(err)
		}
		fmt.Printf("Succefully created dettach request. %q\n", detachment.String())

		b = backoff.NewMaxRetries(maxRetries, retryInterval)
		err = backoff.Retry(o, b)
		if err != nil {
			fmt.Printf("Failed to detach volume after %d retries.\n", maxRetries)
			return microerror.Mask(err)
		}
	}

	fmt.Printf("Volume detached, state %q .\n", ec2.VolumeStateAvailable)
	return nil
}
