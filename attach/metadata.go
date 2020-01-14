package attach

import (
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/giantswarm/microerror"
)

const metadataEndpointInstanceID = "instance-id"

func getInstanceID(session *session.Session) (string, error) {
	ec2metadataClient := ec2metadata.New(session)

	instanceID, err := ec2metadataClient.GetMetadata(metadataEndpointInstanceID)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return instanceID, nil
}
