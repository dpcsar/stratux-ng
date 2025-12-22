package config

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	GDL90 GDL90Config `yaml:"gdl90"`
	Sim   SimConfig   `yaml:"sim"`
	AHRS  AHRSConfig  `yaml:"ahrs"`
	Fan   FanConfig   `yaml:"fan"`
	Web   WebConfig   `yaml:"web"`
}

type FanConfig struct {
	Enable bool `yaml:"enable"`

	// PWMPin is BCM GPIO numbering (matches upstream Stratux).
	PWMPin int `yaml:"pwm_pin"`
	// PWMFrequency is the configured base frequency; upstream default is 64000.
	PWMFrequency int `yaml:"pwm_frequency"`
	// TempTargetC is the CPU temperature target in degrees C; upstream default is 50.
	TempTargetC float64 `yaml:"temp_target_c"`
	// PWMDutyMin is the minimum duty cycle percentage; upstream default is 0.
	PWMDutyMin int `yaml:"pwm_duty_min"`
	// UpdateInterval controls how often duty is recomputed; upstream default is 5s.
	UpdateInterval time.Duration `yaml:"update_interval"`
}

type AHRSConfig struct {
	Enable   bool   `yaml:"enable"`
	I2CBus   int    `yaml:"i2c_bus"`
	IMUAddr  uint16 `yaml:"imu_addr"`
	BaroAddr uint16 `yaml:"baro_addr"`
	// Orientation stores the sensor-to-aircraft mapping.
	// If unset, Stratux-NG uses the sensor's native axes.
	Orientation AHRSOrientationConfig `yaml:"orientation"`
}

type AHRSOrientationConfig struct {
	// ForwardAxis mirrors Stratux's IMU mapping convention: +/-1..+/-3
	// corresponding to sensor X/Y/Z, with sign.
	ForwardAxis int `yaml:"forward_axis"`
	// GravityInSensor is a gravity vector captured while the unit is installed
	// in its in-flight orientation. When present (len==3), Stratux-NG will use
	// it along with ForwardAxis.
	GravityInSensor []float64 `yaml:"gravity_in_sensor"`
}

type WebConfig struct {
	Listen string `yaml:"listen"`
}

type GDL90Config struct {
	Dest     string        `yaml:"dest"`
	Interval time.Duration `yaml:"interval"`
	Record   RecordConfig  `yaml:"record"`
	Replay   ReplayConfig  `yaml:"replay"`
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

// DefaultPath is the canonical appliance config path.
//
// When running as a service, prefer keeping config in /data so it can be
// persisted across updates and be writable for in-place edits.
const DefaultPath = "/data/stratux-ng/config.yaml"

// ResolvePath returns the config path to load.
//
// Resolution order:
//  1. explicit path argument (when non-empty)
//  2. STRATUX_NG_CONFIG environment variable (when non-empty)
//  3. DefaultPath
func ResolvePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path != "" {
		return filepath.Clean(path), nil
	}
	if env := strings.TrimSpace(os.Getenv("STRATUX_NG_CONFIG")); env != "" {
		return filepath.Clean(env), nil
	}
	return DefaultPath, nil
}

// LoadAuto resolves a config path (via ResolvePath) and loads it.
// It returns both the loaded config and the resolved path.
func LoadAuto(path string) (Config, string, error) {
	resolved, err := ResolvePath(path)
	if err != nil {
		return Config{}, "", err
	}
	cfg, err := Load(resolved)
	if err != nil {
		return Config{}, "", err
	}
	return cfg, resolved, nil
}

func Load(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		if te, ok := err.(*yaml.TypeError); ok {
			norm := make([]string, 0, len(te.Errors))
			for _, e := range te.Errors {
				e = strings.TrimSpace(e)
				if strings.HasPrefix(e, "line ") {
					if idx := strings.Index(e, ": "); idx != -1 {
						e = strings.TrimSpace(e[idx+2:])
					}
				}
				if e != "" {
					norm = append(norm, e)
				}
			}
			if len(norm) > 0 {
				return Config{}, fmt.Errorf("config contains unknown fields: %s", strings.Join(norm, "; "))
			}
		}
		return Config{}, err
	}
	// Reject multiple YAML documents; they tend to hide mistakes.
	var extra any
	if err := dec.Decode(&extra); err == nil {
		return Config{}, fmt.Errorf("config must contain a single YAML document")
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

	if cfg.GDL90.Record.Enable {
		if cfg.GDL90.Record.Path == "" {
			return fmt.Errorf("gdl90.record.path is required when gdl90.record.enable is true")
		}
	}

	if cfg.GDL90.Replay.Enable {
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

	// AHRS defaults + validation.
	if cfg.AHRS.I2CBus == 0 {
		cfg.AHRS.I2CBus = 1
	}
	if cfg.AHRS.IMUAddr == 0 {
		cfg.AHRS.IMUAddr = 0x68
	}
	if cfg.AHRS.BaroAddr == 0 {
		cfg.AHRS.BaroAddr = 0x77
	}
	if cfg.AHRS.Orientation.ForwardAxis < -3 || cfg.AHRS.Orientation.ForwardAxis > 3 {
		return fmt.Errorf("ahrs.orientation.forward_axis must be between -3 and 3")
	}
	if cfg.AHRS.Orientation.ForwardAxis == 0 {
		if len(cfg.AHRS.Orientation.GravityInSensor) != 0 {
			return fmt.Errorf("ahrs.orientation.gravity_in_sensor requires ahrs.orientation.forward_axis")
		}
	} else {
		g := cfg.AHRS.Orientation.GravityInSensor
		if len(g) != 0 && len(g) != 3 {
			return fmt.Errorf("ahrs.orientation.gravity_in_sensor must have 3 elements")
		}
	}
	if cfg.AHRS.I2CBus <= 0 {
		return fmt.Errorf("ahrs.i2c_bus must be > 0")
	}
	if cfg.AHRS.IMUAddr > 0x7F {
		return fmt.Errorf("ahrs.imu_addr must be a 7-bit I2C address")
	}
	if cfg.AHRS.BaroAddr > 0x7F {
		return fmt.Errorf("ahrs.baro_addr must be a 7-bit I2C address")
	}

	// Fan defaults + validation.
	if cfg.Fan.PWMPin == 0 {
		cfg.Fan.PWMPin = 18
	}
	if cfg.Fan.PWMFrequency == 0 {
		cfg.Fan.PWMFrequency = 64000
	}
	if cfg.Fan.TempTargetC == 0 {
		cfg.Fan.TempTargetC = 50.0
	}
	if cfg.Fan.UpdateInterval <= 0 {
		cfg.Fan.UpdateInterval = 5 * time.Second
	}
	if cfg.Fan.PWMPin <= 0 {
		return fmt.Errorf("fan.pwm_pin must be > 0")
	}
	if cfg.Fan.PWMFrequency <= 0 {
		return fmt.Errorf("fan.pwm_frequency must be > 0")
	}
	if cfg.Fan.TempTargetC <= 0 {
		return fmt.Errorf("fan.temp_target_c must be > 0")
	}
	if cfg.Fan.PWMDutyMin < 0 || cfg.Fan.PWMDutyMin > 100 {
		return fmt.Errorf("fan.pwm_duty_min must be between 0 and 100")
	}
	if cfg.Fan.UpdateInterval <= 0 {
		return fmt.Errorf("fan.update_interval must be > 0")
	}

	// Web UI defaults + validation (Web UI is always enabled).
	listen := strings.TrimSpace(cfg.Web.Listen)
	if listen == "" {
		listen = ":80"
	}
	// Common UX: accept bare port like "80" and normalize to ":80".
	if isAllDigits(listen) {
		listen = ":" + listen
	}
	// Basic validity: must be host:port or :port.
	if _, _, err := net.SplitHostPort(listen); err != nil {
		return fmt.Errorf("web.listen must be in the form :PORT or HOST:PORT: %w", err)
	}
	cfg.Web.Listen = listen

	return nil
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
