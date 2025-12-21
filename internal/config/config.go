package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	GDL90 GDL90Config `yaml:"gdl90"`
	Sim   SimConfig   `yaml:"sim"`
	Web   WebConfig   `yaml:"web"`
}

type WebConfig struct {
	Enable bool   `yaml:"enable"`
	Listen string `yaml:"listen"`
}

type GDL90Config struct {
	Dest        string        `yaml:"dest"`
	Interval    time.Duration `yaml:"interval"`
	Mode        string        `yaml:"mode"`
	TestPayload string        `yaml:"test_payload"`
	Record      RecordConfig  `yaml:"record"`
	Replay      ReplayConfig  `yaml:"replay"`
}

type RecordConfig struct {
	Enable bool   `yaml:"enable"`
	Path   string `yaml:"path"`
}

type ReplayConfig struct {
	Enable bool    `yaml:"enable"`
	Path   string  `yaml:"path"`
	Speed  float64 `yaml:"speed"`
	Loop   bool    `yaml:"loop"`
}

type SimConfig struct {
	Ownship  OwnshipSimConfig  `yaml:"ownship"`
	Traffic  TrafficSimConfig  `yaml:"traffic"`
	Scenario ScenarioSimConfig `yaml:"scenario"`
}

// ScenarioSimConfig enables deterministic, script-driven simulation.
//
// When enabled, the normal `sim.ownship` and `sim.traffic` generators are
// ignored and frames are built from the scenario script.
type ScenarioSimConfig struct {
	Enable       bool   `yaml:"enable"`
	Path         string `yaml:"path"`
	StartTimeUTC string `yaml:"start_time_utc"`
	Loop         bool   `yaml:"loop"`
}

type TrafficSimConfig struct {
	Enable   bool          `yaml:"enable"`
	Count    int           `yaml:"count"`
	RadiusNm float64       `yaml:"radius_nm"`
	Period   time.Duration `yaml:"period"`
	GroundKt int           `yaml:"ground_kt"`
}

type OwnshipSimConfig struct {
	Enable                 bool          `yaml:"enable"`
	CenterLatDeg           float64       `yaml:"center_lat_deg"`
	CenterLonDeg           float64       `yaml:"center_lon_deg"`
	AltFeet                int           `yaml:"alt_feet"`
	GroundKt               int           `yaml:"ground_kt"`
	GPSHorizontalAccuracyM float64       `yaml:"gps_horizontal_accuracy_m"`
	RadiusNm               float64       `yaml:"radius_nm"`
	Period                 time.Duration `yaml:"period"`
	ICAO                   string        `yaml:"icao"`
	Callsign               string        `yaml:"callsign"`
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
	if err := DefaultAndValidate(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// DefaultAndValidate applies defaults to cfg and validates it.
//
// This is exported so callers (e.g. CLI overrides) can safely mutate a loaded
// config and then re-run the same validation logic as Load().
//
// Error strings are treated as test-stable.
func DefaultAndValidate(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	if cfg.GDL90.Dest == "" {
		return fmt.Errorf("gdl90.dest is required")
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

	if cfg.GDL90.Record.Enable {
		if cfg.GDL90.Mode == "test" {
			return fmt.Errorf("gdl90.record cannot be used with gdl90.mode=test")
		}
		if cfg.GDL90.Record.Path == "" {
			return fmt.Errorf("gdl90.record.path is required when gdl90.record.enable is true")
		}
	}

	if cfg.GDL90.Replay.Enable {
		if cfg.GDL90.Mode == "test" {
			return fmt.Errorf("gdl90.replay cannot be used with gdl90.mode=test")
		}
		if cfg.GDL90.Replay.Path == "" {
			return fmt.Errorf("gdl90.replay.path is required when gdl90.replay.enable is true")
		}
		if cfg.GDL90.Replay.Speed == 0 {
			cfg.GDL90.Replay.Speed = 1
		}
		if cfg.GDL90.Replay.Speed < 0 {
			return fmt.Errorf("gdl90.replay.speed must be > 0")
		}
	}

	if cfg.GDL90.Record.Enable && cfg.GDL90.Replay.Enable {
		return fmt.Errorf("gdl90.record and gdl90.replay cannot both be enabled")
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
	if cfg.Sim.Ownship.GPSHorizontalAccuracyM == 0 {
		// 50m maps to NACp=8 in Stratux.
		cfg.Sim.Ownship.GPSHorizontalAccuracyM = 50
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

	// Scenario defaults + validation.
	if cfg.Sim.Scenario.Enable {
		if cfg.Sim.Scenario.Path == "" {
			return fmt.Errorf("sim.scenario.path is required when sim.scenario.enable is true")
		}
		if strings.TrimSpace(cfg.Sim.Scenario.StartTimeUTC) == "" {
			// Fixed start time keeps scenario runs reproducible.
			cfg.Sim.Scenario.StartTimeUTC = "2020-01-01T00:00:00Z"
		}
		if _, err := time.Parse(time.RFC3339, cfg.Sim.Scenario.StartTimeUTC); err != nil {
			return fmt.Errorf("sim.scenario.start_time_utc must be RFC3339 (e.g. 2020-01-01T00:00:00Z): %w", err)
		}
	}

	// Web UI defaults + validation.
	if strings.TrimSpace(cfg.Web.Listen) == "" {
		cfg.Web.Listen = ":8080"
	}
	if cfg.Web.Enable {
		if strings.TrimSpace(cfg.Web.Listen) == "" {
			return fmt.Errorf("web.listen is required when web.enable is true")
		}
	}

	return nil
}
