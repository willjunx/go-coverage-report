package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

var Default = Config{
	RootPackage: "",
	Threshold: Threshold{
		File:    0,
		Package: 0,
		Total:   0,
	},
	Exclude: Exclude{Paths: nil},
}

func FromFile(cfg *Config, filename string) error {
	source, err := os.ReadFile(filepath.Clean(filename))
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
