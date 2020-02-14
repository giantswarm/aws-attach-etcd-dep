package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/aws-attach-etcd-dep/metadata"
)

func getAWSSession() (*session.Session, error) {
	awsSession := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	region, err := metadata.GetRegion(awsSession)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	fmt.Printf("Fetched region '%s'\n", region)

	// recreate aws session, this time with proper region
	awsSession, err = session.NewSession(&aws.Config{
		Region: aws.String(region)},
	)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	fmt.Printf("Sucesfully created aws client session.\n")
	return awsSession, nil
}
