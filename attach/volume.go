package attach

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"
)

const (
	maxRetries    = 10
	retryInterval = time.Second * 5
)

func (s *Service) getEBSVolumeID(ec2Client *ec2.EC2) (string, error) {
	volume, err := s.describeEBSVolume(ec2Client)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return *volume.VolumeId, nil
}

func (s *Service) describeEBSVolume(ec2Client *ec2.EC2) (*ec2.Volume, error) {
	volumeFilter := &ec2.Filter{
		Name:   tagKey(s.tagKey),
		Values: tagValue(s.tagValue),
	}

	describeVolumeInput := &ec2.DescribeVolumesInput{
		Filters: []*ec2.Filter{
			volumeFilter,
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

func (s *Service) attachEBSVolume(ec2Client *ec2.EC2, instanceID string, volumeID string) error {
	attachVolumeInput := &ec2.AttachVolumeInput{
		Device:     aws.String(s.deviceName),
		InstanceId: aws.String(instanceID),
		VolumeId:   aws.String(volumeID),
	}
	attachment, err := ec2Client.AttachVolume(attachVolumeInput)
	if err != nil {
		return microerror.Mask(err)
	}
	fmt.Printf("Succefully created attach request. %s\n", attachment.String())

	b := backoff.NewMaxRetries(maxRetries, retryInterval)
	o := func() error {
		volume, err := s.describeEBSVolume(ec2Client)
		if err != nil {
			return microerror.Mask(err)
		}

		if *volume.State != ec2.VolumeStateInUse && *volume.Attachments[0].InstanceId == instanceID {
			fmt.Printf("Volume state is '%s', expecting '%s'. Retrying in %ds.\n", *volume.State, ec2.VolumeStateInUse, retryInterval/time.Second)
			return volumeNotAttachedError
		}
		return nil
	}
	err = backoff.Retry(o, b)
	if err != nil {
		fmt.Printf("failed to attach volume after %d retries\n", maxRetries)
		return microerror.Mask(err)
	}

	fmt.Printf("Volume attached, state '%s' .\n", ec2.VolumeStateInUse)
	return nil
}

func (s *Service) detachEBSVolume(ec2Client *ec2.EC2, volume *ec2.Volume) error {
	detachVolumeInput := &ec2.DetachVolumeInput{
		Device:     volume.Attachments[0].Device,
		InstanceId: volume.Attachments[0].InstanceId,
		VolumeId:   volume.VolumeId,
		Force:      aws.Bool(s.forceDetach),
	}

	detachment, err := ec2Client.DetachVolume(detachVolumeInput)
	if err != nil {
		return microerror.Mask(err)
	}
	fmt.Printf("Succefully created dettach request. %s\n", detachment.String())

	b := backoff.NewMaxRetries(maxRetries, retryInterval)
	o := func() error {
		volume, err := s.describeEBSVolume(ec2Client)
		if err != nil {
			return microerror.Mask(err)
		}

		if *volume.State != ec2.VolumeStateAvailable {
			fmt.Printf("Volume state is '%s', expecting '%s'. Retrying in %ds.\n", *volume.State, ec2.VolumeStateAvailable, retryInterval/time.Second)
			return volumeNotAttachedError
		}
		return nil
	}
	err = backoff.Retry(o, b)
	if err != nil {
		fmt.Printf("failed to detach volume after %d retries\n", maxRetries)
		return microerror.Mask(err)
	}

	fmt.Printf("Volume detached, state '%s' .\n", ec2.VolumeStateAvailable)
	return nil
}

func tagKey(input string) *string {
	return aws.String(fmt.Sprintf("tag:%s", input))
}

func tagValue(input string) []*string {
	return []*string{aws.String(input)}
}
