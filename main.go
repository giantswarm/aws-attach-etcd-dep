package main

import (
	"context"
	"fmt"
	"os"

	"github.com/giantswarm/microerror"
	flag "github.com/spf13/pflag"

	"github.com/giantswarm/aws-attach-ebs-by-tag/pkg/project"
)

type Flag struct {
	DeviceName  string
	ForceDetach bool
	TagKey      string
	TagValue    string
}

func main() {
	err := mainError()
	if err != nil {
		panic(fmt.Sprintf("%#v", err))
	}
}

func mainError() error {
	var err error

	var f Flag
	flag.StringVar(&f.DeviceName, "device-name", "/dev/xvdh", "Device name that will be used for attaching the EBS volume.")
	flag.BoolVar(&f.ForceDetach, "force-detach", false, "If set to true, app will use force-detach if the EBS cannot be detached by normal detach operation..")
	flag.StringVar(&f.TagKey, "tag-key", "my-ebs-tag", "Tag key that will be used to found the requested EBS in AWS API.")
	flag.StringVar(&f.TagValue, "tag-value", "test", "Tag value that will be used to found the requested EBS in AWS API, this tag should identify one unique in one EBS.")

	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("%s:%s - %s", project.Name(), project.Version(), project.GitSHA())
		return nil
	}
	if len(os.Args) > 1 && os.Args[1] == "--help" {
		flag.Usage()
		return nil
	}
	flag.Parse()

	// implementation here

	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}
