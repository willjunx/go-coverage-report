package config

import "fmt"

type Config struct {
	RootPackage string
	Threshold   Threshold `yaml:"threshold"`
	Exclude     Exclude   `yaml:"exclude"`
}

type Exclude struct {
	Paths []string `yaml:"paths"`
}

type Threshold struct {
	File    int `yaml:"file"`
	Package int `yaml:"package"`
	Total   int `yaml:"total"`
}

func (c Threshold) validate() error {
	if !inRange(c.File) {
		return fmt.Errorf("file %w", ErrThresholdNotInRange)
	}

	if !inRange(c.Package) {
		return fmt.Errorf("package %w", ErrThresholdNotInRange)
	}

	if !inRange(c.Total) {
		return fmt.Errorf("total %w", ErrThresholdNotInRange)
	}

	return nil
}
