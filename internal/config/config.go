package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	GDL90 GDL90Config `yaml:"gdl90"`
}

type GDL90Config struct {
	Dest        string        `yaml:"dest"`
	Interval    time.Duration `yaml:"interval"`
	Mode        string        `yaml:"mode"`
	TestPayload string        `yaml:"test_payload"`
}

func Load(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return Config{}, err
	}

	if cfg.GDL90.Dest == "" {
		return Config{}, fmt.Errorf("gdl90.dest is required")
	}
	if cfg.GDL90.Interval <= 0 {
		cfg.GDL90.Interval = 1 * time.Second
	}
	if cfg.GDL90.Mode == "" {
		cfg.GDL90.Mode = "gdl90"
	}
	if cfg.GDL90.TestPayload == "" {
		cfg.GDL90.TestPayload = "STRATUX-NG TEST"
	}

	return cfg, nil
}
