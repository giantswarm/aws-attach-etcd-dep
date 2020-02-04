package eni

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"
)

const (
	maxRetries    = 15
	retryInterval = time.Second * 10
)

func (s *Service) getEniID(ec2Client *ec2.EC2) (string, error) {
	eni, err := s.describeEni(ec2Client)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return *eni.NetworkInterfaceId, nil
}

func (s *Service) describeEni(ec2Client *ec2.EC2) (*ec2.NetworkInterface, error) {
	eniFilter := &ec2.Filter{
		Name:   tagKey(s.tagKey),
		Values: tagValue(s.tagValue),
	}

	describeVolumeInput := &ec2.DescribeNetworkInterfacesInput{
		Filters: []*ec2.Filter{
			eniFilter,
		},
	}
	o, err := ec2Client.DescribeNetworkInterfaces(describeVolumeInput)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// tags should give us only one unique volume
	if len(o.NetworkInterfaces) != 1 {
		return nil, microerror.Maskf(executionFailedError, "expected 1 eni but got %d instead", len(o.NetworkInterfaces))
	}

	return o.NetworkInterfaces[0], nil
}

func (s *Service) attachEni(ec2Client *ec2.EC2, instanceID string, eniID string) error {
	attachNetworkInterfaceInput := &ec2.AttachNetworkInterfaceInput{
		DeviceIndex:        aws.Int64(s.deviceIndex),
		InstanceId:         aws.String(instanceID),
		NetworkInterfaceId: aws.String(eniID),
	}
	attachment, err := ec2Client.AttachNetworkInterface(attachNetworkInterfaceInput)
	if err != nil {
		return microerror.Mask(err)
	}
	fmt.Printf("Succefully created attach request. %s\n", attachment.String())

	b := backoff.NewMaxRetries(maxRetries, retryInterval)
	o := func() error {
		eni, err := s.describeEni(ec2Client)
		if err != nil {
			return microerror.Mask(err)
		}

		if *eni.Status != ec2.NetworkInterfaceStatusInUse && *eni.Attachment.InstanceId == instanceID {
			fmt.Printf("Volume state is '%s', expecting '%s', retrying in %ds.\n", *eni.Status, ec2.NetworkInterfaceStatusInUse, retryInterval/time.Second)
			return eniNotAttachedError
		}
		return nil
	}
	err = backoff.Retry(o, b)
	if err != nil {
		fmt.Printf("failed to attach eni after %d retries\n", maxRetries)
		return microerror.Mask(err)
	}

	fmt.Printf("ENI attached, state '%s' .\n", ec2.NetworkInterfaceStatusInUse)
	return nil
}

func (s *Service) detachEni(ec2Client *ec2.EC2, eni *ec2.NetworkInterface) error {
	detachNetworkInterfaceInput := &ec2.DetachNetworkInterfaceInput{
		AttachmentId: eni.Attachment.AttachmentId,
		Force:        aws.Bool(s.forceDetach),
	}

	detachment, err := ec2Client.DetachNetworkInterface(detachNetworkInterfaceInput)
	if err != nil {
		return microerror.Mask(err)
	}
	fmt.Printf("Succefully created dettach request. %s\n", detachment.String())

	b := backoff.NewMaxRetries(maxRetries, retryInterval)
	o := func() error {
		eni, err := s.describeEni(ec2Client)
		if err != nil {
			return microerror.Mask(err)
		}

		if *eni.Status != ec2.NetworkInterfaceStatusAvailable {
			fmt.Printf("Volume state is '%s', expecting '%s', retrying in %ds.\n", *eni.Status, ec2.NetworkInterfaceStatusAvailable, retryInterval/time.Second)
			return eniNotDetachedError
		}
		return nil
	}
	err = backoff.Retry(o, b)
	if err != nil {
		fmt.Printf("failed to detach eni after %d retries\n", maxRetries)
		return microerror.Mask(err)
	}

	fmt.Printf("ENI detached, state '%s' .\n", ec2.NetworkInterfaceStatusAvailable)
	return nil
}

func tagKey(input string) *string {
	return aws.String(fmt.Sprintf("tag:%s", input))
}

func tagValue(input string) []*string {
	return []*string{aws.String(input)}
}
