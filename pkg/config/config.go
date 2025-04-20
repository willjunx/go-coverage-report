package config

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

var Default = Config{
	Threshold: Threshold{
		File:    0,
		Package: 0,
		Total:   0,
	},
}

func ConfigFromFile(cfg *Config, filename string) error {
	source, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed reading file: %w", err)
	}

	err = yaml.Unmarshal(source, cfg)
	if err != nil {
		return fmt.Errorf("failed parsing config file: %w", err)
	}

	return errors.Join(
		cfg.Threshold.validate(),
	)
}
