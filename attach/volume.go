package attach

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/giantswarm/microerror"
)

func (s *Service) getEBSVolumeID(ec2Client *ec2.EC2) (string, error) {

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
		return "", microerror.Mask(err)
	}

	// tags should give us only one unique volume
	if len(o.Volumes) != 1 {
		return "", microerror.Maskf(executionFailedError, "expected 1 volume but got %d instead", len(o.Volumes))
	}

	// fetch the volume ID
	volumeID := *o.Volumes[0].VolumeId
	return volumeID, nil
}

func tagKey(input string) *string {
	return aws.String(fmt.Sprintf("Tag:%s", input))
}

func tagValue(input string) []*string {
	return []*string{aws.String(input)}
}
