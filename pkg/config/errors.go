package config

import "errors"

var (
	ErrThresholdNotInRange = errors.New("threshold must be in range [0 - 100]")
)
