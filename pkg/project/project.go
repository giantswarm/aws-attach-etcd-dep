package project

var (
	description        = "The aws-attach-ebs-by-tag attach EBS volume to the ec2 instance specified by tag."
	gitSHA             = "n/a"
	name        string = "aws-attach-ebs-by-tag"
	source      string = "https://github.com/giantswarm/aws-attach-ebs-by-tag"
	version            = "0.4.0"
)

func Description() string {
	return description
}

func GitSHA() string {
	return gitSHA
}

func Name() string {
	return name
}

func Source() string {
	return source
}

func Version() string {
	return version
}
