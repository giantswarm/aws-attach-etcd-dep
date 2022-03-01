package aws

import (
	"fmt"
	"net"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/aws-attach-etcd-dep/routing"
)

type ENIConfig struct {
	AWSInstanceID string
	AwsSession    *session.Session
	DeviceIndex   int64
	ForceDetach   bool
	TagKey        string
	TagValue      string
}

type ENI struct {
	awsInstanceID string
	awsSession    *session.Session
	deviceIndex   int64
	forceDetach   bool
	tagKey        string
	tagValue      string
}

func NewENI(config ENIConfig) (*ENI, error) {
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

	newENI := &ENI{
		awsInstanceID: config.AWSInstanceID,
		awsSession:    config.AwsSession,
		deviceIndex:   config.DeviceIndex,
		forceDetach:   config.ForceDetach,
		tagKey:        config.TagKey,
		tagValue:      config.TagValue,
	}
	return newENI, nil
}

func (s *ENI) AttachByTag() error {
	// create ec2 client
	ec2Client := ec2.New(s.awsSession)

	eni, err := s.describe(ec2Client)
	if err != nil {
		return microerror.Mask(err)
	}
	fmt.Printf("Fetched eni-id '%s'\n", *eni.NetworkInterfaceId)

	if *eni.Status == ec2.NetworkInterfaceStatusInUse &&
		*eni.Attachment.InstanceId == s.awsInstanceID {
		fmt.Printf("ENI is already attached to this instance. Nothing to do.\n")
		return nil
	} else if *eni.Status == ec2.NetworkInterfaceStatusInUse {
		fmt.Printf("ENI is attached to %q and is in state %q. Trying detach the volume\n", *eni.Attachment.InstanceId, *eni.Status)

		err := s.detach(ec2Client, eni)
		if err != nil {
			return microerror.Mask(err)
		}
	} else {
		fmt.Printf("ENI state is %q.\n", *eni.Status)
	}

	err = s.attach(ec2Client, s.awsInstanceID, *eni.NetworkInterfaceId)
	if err != nil {
		return microerror.Mask(err)
	}

	awsEniSubnet, err := s.describeSubnet(ec2Client, *eni.SubnetId)
	if err != nil {
		return microerror.Mask(err)
	}

	_, ipNet, err := net.ParseCIDR(*awsEniSubnet.CidrBlock)
	if err != nil {
		return microerror.Mask(err)
	}

	err = routing.ConfigureNetworkRoutingForENI(*eni.PrivateIpAddress, ipNet)
	if err != nil {
		return microerror.Mask(err)
	}
	fmt.Printf("Sucesfully configured routing for eth1  for ip %s.\n", *eni.PrivateIpAddress)
	return nil
}

func (s *ENI) describe(ec2Client *ec2.EC2) (*ec2.NetworkInterface, error) {
	eniFilter := &ec2.Filter{
		Name:   tagKey(s.tagKey),
		Values: tagValue(s.tagValue),
	}

	describeVolumeInput := &ec2.DescribeNetworkInterfacesInput{
		Filters: []*ec2.Filter{
			eniFilter,
		},
	}
	var eni *ec2.NetworkInterface
	b := backoff.NewMaxRetries(maxRetries, retryInterval)
	o := func() error {
		out, err := ec2Client.DescribeNetworkInterfaces(describeVolumeInput)
		if err != nil {
			return microerror.Mask(err)
		}

		// tags should give us only one unique volume
		if len(out.NetworkInterfaces) != 1 {
			fmt.Printf("expected 1 eni but got %d instead retrying in %ds\n", len(out.NetworkInterfaces), retryInterval/time.Second)
			return microerror.Maskf(executionFailedError, "expected 1 eni but got %d instead", len(out.NetworkInterfaces))
		}

		eni = out.NetworkInterfaces[0]
		return nil
	}
	err := backoff.Retry(o, b)
	if err != nil {
		fmt.Printf("Failed to describe eni after %d retries.\n", maxRetries)
		return nil, microerror.Mask(err)
	}

	return eni, nil
}

func (s *ENI) attach(ec2Client *ec2.EC2, instanceID string, eniID string) error {
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
		eni, err := s.describe(ec2Client)
		if err != nil {
			return microerror.Mask(err)
		}

		if *eni.Status != ec2.NetworkInterfaceStatusInUse && *eni.Attachment.InstanceId == instanceID {
			fmt.Printf("ENI state is %q, expecting %q, retrying in %ds.\n", *eni.Status, ec2.NetworkInterfaceStatusInUse, retryInterval/time.Second)
			return microerror.Maskf(executionFailedError, "ENI not attached")
		}
		return nil
	}
	err = backoff.Retry(o, b)
	if err != nil {
		fmt.Printf("Failed to attach eni after %d retries.\n", maxRetries)
		return microerror.Mask(err)
	}

	fmt.Printf("ENI attached, state %q .\n", ec2.NetworkInterfaceStatusInUse)
	return nil
}

func (s *ENI) detach(ec2Client *ec2.EC2, eni *ec2.NetworkInterface) error {
	// wait if automatic detach happens  by terminating the instance

	b := backoff.NewMaxRetries(waitAutoDetachMaxRetries, retryInterval)
	o := func() error {
		eni, err := s.describe(ec2Client)
		if err != nil {
			return microerror.Mask(err)
		}

		if *eni.Status != ec2.NetworkInterfaceStatusAvailable {
			fmt.Printf("ENI state is %q, expecting %q, retrying in %s.\n", *eni.Status, ec2.NetworkInterfaceStatusAvailable, retryInterval)
			return microerror.Maskf(executionFailedError, "ENI not detached")
		}
		return nil
	}

	err := backoff.Retry(o, b)
	if err == nil {
		// the ENI was eventually detached by the instance by itself, no need for manual detach
		return nil
	} else {
		// eni is still attached after 10mins, lets try detach it manually here
		detachNetworkInterfaceInput := &ec2.DetachNetworkInterfaceInput{
			AttachmentId: eni.Attachment.AttachmentId,
			Force:        aws.Bool(s.forceDetach),
		}

		detachment, err := ec2Client.DetachNetworkInterface(detachNetworkInterfaceInput)
		if err != nil {
			return microerror.Mask(err)
		}
		fmt.Printf("Succefully created dettach request. %s\n", detachment.String())

		b = backoff.NewMaxRetries(maxRetries, retryInterval)
		err = backoff.Retry(o, b)
		if err != nil {
			fmt.Printf("Failed to detach eni after %d retries.\n", maxRetries)
			return microerror.Mask(err)
		}
	}
	fmt.Printf("ENI detached, state %q .\n", ec2.NetworkInterfaceStatusAvailable)
	return nil
}

func (s *ENI) describeSubnet(ec2Client *ec2.EC2, subnetID string) (*ec2.Subnet, error) {
	describeSubnetInput := &ec2.DescribeSubnetsInput{
		SubnetIds: []*string{aws.String(subnetID)},
	}
	o, err := ec2Client.DescribeSubnets(describeSubnetInput)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// id should give us only one unique subnet
	if len(o.Subnets) != 1 {
		return nil, microerror.Maskf(executionFailedError, "expected 1 eni for subnedID %#q but got %d instead", subnetID, len(o.Subnets))
	}

	return o.Subnets[0], nil
}
