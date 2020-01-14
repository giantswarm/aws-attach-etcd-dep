package attach

import "github.com/giantswarm/microerror"

type Config struct {
	DeviceName  string
	ForceDetach bool
	TagKey      string
	TagValue    string
}

type Service struct {
	deviceName  string
	forceDetach bool
	tagKey      string
	tagValue    string
}

func New(config Config) (*Service, error) {
	if config.DeviceName == "" {
		return nil, microerror.Maskf(invalidConfigError, "config.DeviceName must not be empty")
	}
	if config.TagKey == "" {
		return nil, microerror.Maskf(invalidConfigError, "config.TagKey must not be empty")
	}
	if config.TagValue == "" {
		return nil, microerror.Maskf(invalidConfigError, "config.TagValue must not be empty")
	}

	newService:=  &Service{
		deviceName: config.DeviceName,
		forceDetach: config.ForceDetach,
		tagKey:      config.TagKey,
		tagValue:    config.TagValue,
	}
	return newService, nil
}

func (s *Service) AttachEBSByTag() error {

	return nil
}