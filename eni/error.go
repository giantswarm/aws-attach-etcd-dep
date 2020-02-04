package eni

import "github.com/giantswarm/microerror"

var invalidConfigError = microerror.New("invalid config error")

var executionFailedError = microerror.New("execution failed error")

var eniNotAttachedError = microerror.New("eni not attached error")

var eniNotDetachedError = microerror.New("eni not detached error")
