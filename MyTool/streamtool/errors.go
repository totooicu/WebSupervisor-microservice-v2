package streamtool

import "errors"

var (
	ErrResponseChannelNotFound = errors.New("response channel not found")
	ErrResponseTimeout        = errors.New("response timeout")
	ErrServiceNotFound        = errors.New("service not found")
	ErrStreamPushFailed       = errors.New("stream push failed")
	ErrStreamGetFailed        = errors.New("stream get failed")
)