package config

import "errors"

// ErrConfigFileNotFound is used as a sentinel error to distinguish when a docker config file is not present,
// so that cases without a config file work, but cases with an invalid config file still fail
var ErrConfigFileNotFound = errors.New("config file not found")
