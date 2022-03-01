[![CircleCI](https://circleci.com/gh/giantswarm/aws-attach-etcd-dep/tree/master.svg?style=svg)](https://circleci.com/gh/giantswarm/aws-attach-etcd-dep/tree/master)

# aws attach etcd dependencies

this utility will:
* atach EBS volume to the instance specified by tag
* create necessery fs on the EBS volume if not already formatted
* attach ENI to the instance specified by tag

implemented in go with AWS SDK for go
