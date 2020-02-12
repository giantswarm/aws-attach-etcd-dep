[![CircleCI](https://circleci.com/gh/giantswarm/aws-attach-ebs-by-tag.svg?style=shield&circle-token=cbabd7d13186f190fca813db4f0c732b026f5f6c)](https://circleci.com/gh/giantswarm/aws-attach-ebs-by-tag)

# aws attach etcd dependencies

this utility will:
* atach EBS volume to the instance specified by tag
* create necessery fs on the EBS volume if not already formatted
* attach ENI to the instance specified by tag

implemented in go with AWS SDK for go
