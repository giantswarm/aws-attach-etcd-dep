package aws

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
)

const (
	maxRetries    = 15
	retryInterval = time.Second * 10
)

func tagKey(input string) *string {
	return aws.String(fmt.Sprintf("tag:%s", input))
}

func tagValue(input string) []*string {
	return []*string{aws.String(input)}
}
