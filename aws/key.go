package aws

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
)

const (
	// back off retry config
	// retry for 2 hours
	maxRetries    = 240
	retryInterval = time.Second * 15
)

func tagKey(input string) *string {
	return aws.String(fmt.Sprintf("tag:%s", input))
}

func tagValue(input string) []*string {
	return []*string{aws.String(input)}
}
