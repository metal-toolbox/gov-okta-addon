package natslock

import "errors"

// ErrBadParameter is returned when bad parameters are passed to a request
var ErrBadParameter = errors.New("bad parameters in request")
