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
	GDL90   GDL90Config   `yaml:"gdl90"`
	GPS     GPSConfig     `yaml:"gps"`
	Ownship OwnshipConfig `yaml:"ownship"`
	AHRS    AHRSConfig    `yaml:"ahrs"`
	Fan     FanConfig     `yaml:"fan"`
	Web     WebConfig     `yaml:"web"`
	WiFi    WiFiConfig    `yaml:"wifi"`

	// External decoder inputs (planned): 1090 and 978.
	//  - 978 is typically ingested as NDJSON-over-TCP (dump978-fa JSON stream).
	//  - 1090 is ingested by polling a JSON file (dump1090-fa aircraft.json).
	ADSB1090 DecoderBandConfig `yaml:"adsb1090"`
	UAT978   DecoderBandConfig `yaml:"uat978"`
}

// DecoderBandConfig describes one RF band ingest path (e.g. 1090 or 978).
//
// Stratux-NG treats decoders as external processes and ingests their output via
// either:
//   - newline-delimited JSON (NDJSON) over TCP, or
//   - a periodically-updated JSON file.
type DecoderBandConfig struct {
	Enable bool `yaml:"enable"`

	Decoder DecoderConfig `yaml:"decoder"`
	SDR     SDRSelector   `yaml:"sdr"`
}

// DecoderConfig configures either:
// - a supervised local decoder process (Command non-empty), or
// - an externally-managed decoder (Command empty) that Stratux-NG connects to.
//
// Supported ingest sources:
// - NDJSON-over-TCP via JSONListen/JSONAddr (e.g. dump978-fa --json-port)
// - periodically-updated JSON files via JSONFile (e.g. dump1090-fa aircraft.json)
// - raw line-over-TCP via RawListen/RawAddr (e.g. dump978-fa --raw-port)
//
// For NDJSON ingest, set exactly one of JSONListen, JSONAddr, or JSONFile.
// For raw ingest, set exactly one of RawListen or RawAddr.
//
// At least one ingest source must be configured when the band is enabled.
type DecoderConfig struct {
	Command string   `yaml:"command"`
	Args    []string `yaml:"args"`

	JSONListen string `yaml:"json_listen"`
	JSONAddr   string `yaml:"json_addr"`
	JSONFile   string `yaml:"json_file"`

	// RawListen/RawAddr configure a TCP endpoint that emits newline-delimited
	// dump978-style raw messages (e.g. "+<hex>;rs=...;ss=...;").
	RawListen string `yaml:"raw_listen"`
	RawAddr   string `yaml:"raw_addr"`

	// JSONFileInterval controls how often JSONFile is polled.
	// When zero, defaults to 1s.
	JSONFileInterval time.Duration `yaml:"json_file_interval"`
}

// SDRSelector describes how to select an SDR device.
//
// For RTL-SDR-class devices, the recommended approach is programming a unique
// EEPROM serial string (e.g. stratux:1090, stratux:978) and selecting by it.
type SDRSelector struct {
	// SerialTag is a stable identifier such as "stratux:1090" or "stratux:978".
	SerialTag string `yaml:"serial_tag"`

	// Index is a fallback device index when no serial tag is available.
	Index *int `yaml:"index"`

	// Path is an optional stable device path (e.g. a udev /dev/ symlink).
	Path string `yaml:"path"`
}

type GPSConfig struct {
	Enable bool `yaml:"enable"`

	// Source selects how GPS data is ingested.
	//
	// Supported values:
	// - "nmea": read NMEA sentences directly from a serial device
	// - "gpsd": connect to gpsd and consume JSON reports
	//
	// When empty, defaults to "nmea".
	Source string `yaml:"source"`

	// GPSDAddr is the host:port of a local gpsd instance (default: 127.0.0.1:2947).
	// Only used when Source == "gpsd".
	GPSDAddr string `yaml:"gpsd_addr"`

	// Device is the serial device path (e.g. /dev/ttyACM0 or /dev/ttyUSB0).
	// When empty, Stratux-NG will attempt to auto-detect a likely device.
	Device string `yaml:"device"`

	// Baud is the serial baud rate. Most USB u-blox receivers default to 9600.
	Baud int `yaml:"baud"`

	// HorizontalAccuracyM is used to derive NACp similarly to upstream Stratux.
	HorizontalAccuracyM float64 `yaml:"horizontal_accuracy_m"`
}

type FanConfig struct {
	Enable bool `yaml:"enable"`

	// Backend selects the fan control backend.
	// Supported values:
	// - "" or "pwm": drive /sys/class/pwm (default)
	// - "gpio": drive GPIO pin as on/off (libgpiod)
	Backend string `yaml:"backend"`

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

type OwnshipConfig struct {
	ICAO     string `yaml:"icao"`
	Callsign string `yaml:"callsign"`
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

	// Decoder band defaults / validation. Keep permissive for bring-up:
	// - When Enable=false, ignore configuration.
	// - When Enable=true, require at least one ingest source.
	validateBand := func(name string, b *DecoderBandConfig) error {
		if b == nil || !b.Enable {
			return nil
		}
		listen := strings.TrimSpace(b.Decoder.JSONListen)
		addr := strings.TrimSpace(b.Decoder.JSONAddr)
		file := strings.TrimSpace(b.Decoder.JSONFile)
		rawListen := strings.TrimSpace(b.Decoder.RawListen)
		rawAddr := strings.TrimSpace(b.Decoder.RawAddr)

		jsonSet := 0
		if listen != "" {
			jsonSet++
		}
		if addr != "" {
			jsonSet++
		}
		if file != "" {
			jsonSet++
		}
		rawSet := 0
		if rawListen != "" {
			rawSet++
		}
		if rawAddr != "" {
			rawSet++
		}
		if jsonSet == 0 && rawSet == 0 {
			return fmt.Errorf("%s.decoder must set at least one ingest source (json_* or raw_*)", name)
		}
		if jsonSet != 0 && jsonSet != 1 {
			return fmt.Errorf("%s.decoder must set exactly one of json_listen, json_addr, or json_file", name)
		}
		if rawSet != 0 && rawSet != 1 {
			return fmt.Errorf("%s.decoder must set exactly one of raw_listen or raw_addr", name)
		}
		if rawSet > 0 && name != "uat978" {
			return fmt.Errorf("%s.decoder raw_* is only supported for uat978", name)
		}
		if name == "adsb1090" {
			// Be strict: 1090 ingest is dump1090-fa aircraft.json polling.
			if strings.TrimSpace(b.Decoder.JSONFile) == "" {
				return fmt.Errorf("adsb1090.decoder.json_file is required (dump1090-fa aircraft.json polling)")
			}
			if strings.TrimSpace(b.Decoder.JSONListen) != "" || strings.TrimSpace(b.Decoder.JSONAddr) != "" {
				return fmt.Errorf("adsb1090.decoder only supports json_file (dump1090-fa); json_listen/json_addr are not supported")
			}
		}
		if listen != "" {
			if _, err := net.ResolveTCPAddr("tcp", listen); err != nil {
				return fmt.Errorf("%s.decoder.json_listen invalid: %w", name, err)
			}
		}
		if addr != "" {
			if _, err := net.ResolveTCPAddr("tcp", addr); err != nil {
				return fmt.Errorf("%s.decoder.json_addr invalid: %w", name, err)
			}
		}
		if file != "" {
			// Be permissive: allow relative paths (dev) and non-existent files at startup.
			if strings.ContainsRune(file, '\x00') {
				return fmt.Errorf("%s.decoder.json_file invalid", name)
			}
			if b.Decoder.JSONFileInterval < 0 {
				return fmt.Errorf("%s.decoder.json_file_interval must be >= 0", name)
			}
			if b.Decoder.JSONFileInterval == 0 {
				b.Decoder.JSONFileInterval = 1 * time.Second
			}
		}
		if rawListen != "" {
			if _, err := net.ResolveTCPAddr("tcp", rawListen); err != nil {
				return fmt.Errorf("%s.decoder.raw_listen invalid: %w", name, err)
			}
		}
		if rawAddr != "" {
			if _, err := net.ResolveTCPAddr("tcp", rawAddr); err != nil {
				return fmt.Errorf("%s.decoder.raw_addr invalid: %w", name, err)
			}
		}
		// If we are supervising a decoder, a command is required.
		if strings.TrimSpace(b.Decoder.Command) == "" {
			// external decoder allowed
			return nil
		}
		return nil
	}
	if err := validateBand("adsb1090", &cfg.ADSB1090); err != nil {
		return err
	}
	if err := validateBand("uat978", &cfg.UAT978); err != nil {
		return err
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

	// GPS defaults + validation.
	if strings.TrimSpace(cfg.GPS.Source) == "" {
		cfg.GPS.Source = "nmea"
	}
	cfg.GPS.Source = strings.ToLower(strings.TrimSpace(cfg.GPS.Source))
	if cfg.GPS.Source != "nmea" && cfg.GPS.Source != "gpsd" {
		return fmt.Errorf("gps.source must be one of: nmea, gpsd")
	}
	if cfg.GPS.Source == "gpsd" {
		if strings.TrimSpace(cfg.GPS.GPSDAddr) == "" {
			cfg.GPS.GPSDAddr = "127.0.0.1:2947"
		}
		if _, _, err := net.SplitHostPort(strings.TrimSpace(cfg.GPS.GPSDAddr)); err != nil {
			return fmt.Errorf("gps.gpsd_addr must be host:port")
		}
	}
	if cfg.GPS.Baud == 0 {
		cfg.GPS.Baud = 9600
	}
	if cfg.GPS.Baud < 0 {
		return fmt.Errorf("gps.baud must be > 0")
	}
	if cfg.GPS.HorizontalAccuracyM == 0 {
		cfg.GPS.HorizontalAccuracyM = 10
	}
	if cfg.GPS.HorizontalAccuracyM < 0 {
		return fmt.Errorf("gps.horizontal_accuracy_m must be >= 0")
	}

	if strings.TrimSpace(cfg.Ownship.ICAO) == "" {
		cfg.Ownship.ICAO = "F00000"
	}
	if strings.TrimSpace(cfg.Ownship.Callsign) == "" {
		cfg.Ownship.Callsign = "STRATUX"
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
	backend := strings.TrimSpace(strings.ToLower(cfg.Fan.Backend))
	if backend == "" {
		backend = "auto"
	}
	if backend != "" && backend != "pwm" && backend != "gpio" && backend != "auto" {
		return fmt.Errorf("fan.backend must be one of: auto, pwm, gpio")
	}
	cfg.Fan.Backend = backend
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

type WiFiConfig struct {
	APSSID     string `yaml:"ap_ssid"`
	APPass     string `yaml:"ap_pass"`
	APIP       string `yaml:"ap_ip"`
	ClientSSID string `yaml:"client_ssid"`
	ClientPass string `yaml:"client_pass"`
}
