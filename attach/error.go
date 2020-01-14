package attach

import "github.com/giantswarm/microerror"

var invalidConfigError = microerror.New("invalid config error")

var executionFailedError = microerror.New("execution failed error")

var volumeNotAttachedError = microerror.New("volume not attached error")
