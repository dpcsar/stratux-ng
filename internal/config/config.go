package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	GDL90 GDL90Config `yaml:"gdl90"`
	Sim   SimConfig   `yaml:"sim"`
}

type GDL90Config struct {
	Dest        string        `yaml:"dest"`
	Interval    time.Duration `yaml:"interval"`
	Mode        string        `yaml:"mode"`
	TestPayload string        `yaml:"test_payload"`
}

type SimConfig struct {
	Ownship OwnshipSimConfig `yaml:"ownship"`
	Traffic TrafficSimConfig `yaml:"traffic"`
}

type TrafficSimConfig struct {
	Enable   bool          `yaml:"enable"`
	Count    int           `yaml:"count"`
	RadiusNm float64       `yaml:"radius_nm"`
	Period   time.Duration `yaml:"period"`
	GroundKt int           `yaml:"ground_kt"`
}

type OwnshipSimConfig struct {
	Enable       bool          `yaml:"enable"`
	CenterLatDeg float64       `yaml:"center_lat_deg"`
	CenterLonDeg float64       `yaml:"center_lon_deg"`
	AltFeet      int           `yaml:"alt_feet"`
	GroundKt     int           `yaml:"ground_kt"`
	RadiusNm     float64       `yaml:"radius_nm"`
	Period       time.Duration `yaml:"period"`
	ICAO         string        `yaml:"icao"`
	Callsign     string        `yaml:"callsign"`
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

	// Simulator defaults (safe even if disabled).
	if cfg.Sim.Ownship.Period <= 0 {
		cfg.Sim.Ownship.Period = 120 * time.Second
	}
	if cfg.Sim.Ownship.RadiusNm <= 0 {
		cfg.Sim.Ownship.RadiusNm = 0.5
	}
	if cfg.Sim.Ownship.GroundKt <= 0 {
		cfg.Sim.Ownship.GroundKt = 90
	}
	if cfg.Sim.Ownship.AltFeet == 0 {
		cfg.Sim.Ownship.AltFeet = 3000
	}
	if cfg.Sim.Ownship.ICAO == "" {
		cfg.Sim.Ownship.ICAO = "F00000"
	}
	if cfg.Sim.Ownship.Callsign == "" {
		cfg.Sim.Ownship.Callsign = "STRATUX"
	}

	// Traffic simulator defaults.
	if cfg.Sim.Traffic.Count <= 0 {
		cfg.Sim.Traffic.Count = 3
	}
	if cfg.Sim.Traffic.RadiusNm <= 0 {
		cfg.Sim.Traffic.RadiusNm = 2.0
	}
	if cfg.Sim.Traffic.Period <= 0 {
		cfg.Sim.Traffic.Period = 90 * time.Second
	}
	if cfg.Sim.Traffic.GroundKt <= 0 {
		cfg.Sim.Traffic.GroundKt = 120
	}

	return cfg, nil
}
